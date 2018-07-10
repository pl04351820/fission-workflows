package invocation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fission/fission-workflows/pkg/api"
	"github.com/fission/fission-workflows/pkg/controller"
	"github.com/fission/fission-workflows/pkg/controller/expr"
	"github.com/fission/fission-workflows/pkg/fes"
	"github.com/fission/fission-workflows/pkg/scheduler"
	"github.com/fission/fission-workflows/pkg/types/aggregates"
	"github.com/fission/fission-workflows/pkg/types/events"
	"github.com/fission/fission-workflows/pkg/util/gopool"
	"github.com/fission/fission-workflows/pkg/util/labels"
	"github.com/fission/fission-workflows/pkg/util/pubsub"
	"github.com/golang/protobuf/ptypes"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	NotificationBuffer    = 100
	Name                  = "invocation"
	maxParallelExecutions = 1000
)

var (
	wfiLog = log.WithField("component", "controller.invocation")

	// workflow-related metrics
	invocationStatus = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "workflows",
		Subsystem: "controller_invocation",
		Name:      "status",
		Help:      "Count of the different statuses of workflow invocations.",
	}, []string{"status"})

	invocationDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "workflows",
		Subsystem: "controller_invocation",
		Name:      "finished_duration",
		Help:      "Duration of an invocation from start to a finished state.",
	})

	exprEvalDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "workflows",
		Subsystem: "controller_invocation",
		Name:      "expr_eval_duration",
		Help:      "Duration of the evaluation of the input expressions.",
	})
)

func init() {
	prometheus.MustRegister(invocationStatus, invocationDuration, exprEvalDuration)
}

// var wfiLog *log.Entry

// func init() {
// 	l := log.New()
// 	l.SetLevel(log.DebugLevel)
// 	wfiLog = l.WithField("component", "controller-wi")
// }

type Controller struct {
	invokeCache   fes.CacheReader
	wfCache       fes.CacheReader
	taskAPI       *api.Task
	invocationAPI *api.Invocation
	cancelFn      context.CancelFunc
	evalStates    *expr.Store
	evalPolicy    controller.Rule
	evalStore     controller.EvalStore
	evalQueue     *controller.ConcurrentEvalStateHeap
	fesSub        *pubsub.Subscription
	scheduler     *scheduler.WorkflowScheduler
}

func NewController(invokeCache fes.CacheReader, wfCache fes.CacheReader, workflowScheduler *scheduler.WorkflowScheduler,
	taskAPI *api.Task, invocationAPI *api.Invocation, evalStates *expr.Store) *Controller {
	ctr := &Controller{
		invokeCache:   invokeCache,
		wfCache:       wfCache,
		scheduler:     workflowScheduler,
		taskAPI:       taskAPI,
		invocationAPI: invocationAPI,
		// evalQueue:     make(chan string, defaultEvalQueueSize),
		evalStates: evalStates,
		// invokeCache: invokeCache,
		// wfCache:     wfCache,
		// scheduler:   workflowScheduler,

		// States maintains an active cache of currently running invocations, with execution related data.
		// This state information is considered preemptable and can be removed or lost at any time.
		//states: map[string]*ControlState{},
		evalQueue: controller.NewConcurrentEvalStateHeap(true),
	}

	ctr.evalPolicy = defaultPolicy(ctr)
	return ctr
}

func (cr *Controller) Init(sctx context.Context) error {
	ctx, cancelFn := context.WithCancel(sctx)
	cr.cancelFn = cancelFn

	// Subscribe to invocation creations and task events.
	selector := labels.In(fes.PubSubLabelAggregateType, "invocation", "function")
	if invokePub, ok := cr.invokeCache.(pubsub.Publisher); ok {
		cr.fesSub = invokePub.Subscribe(pubsub.SubscriptionOptions{
			Buffer:       NotificationBuffer,
			LabelMatcher: selector,
		})

		// Invocation Notification listener
		go func(ctx context.Context) {
			for {
				select {
				case notification := <-cr.fesSub.Ch:
					cr.handleMsg(notification)
				case <-ctx.Done():
					wfiLog.Debug("Notification listener stopped.")
					return
				}
			}
		}(ctx)
	}

	// process evaluation queue
	pool := gopool.New(maxParallelExecutions)
	go func(ctx context.Context) {
		queue := cr.evalQueue.Chan()
		for {
			select {
			case eval := <-queue:
				err := pool.Submit(ctx, func() {
					controller.EvalQueueSize.WithLabelValues("invocation").Dec()
					cr.Evaluate(eval.ID())
				})
				if err != nil {
					wfiLog.Errorf("failed to submit invocation %v for execution", eval.ID())
				}
			case <-ctx.Done():
				wfiLog.Debug("Evaluation queue listener stopped.")
				return
			}
		}
	}(ctx)

	return nil
}

func (cr *Controller) handleMsg(msg pubsub.Msg) error {
	switch n := msg.(type) {
	case *fes.Notification:
		cr.Notify(n)
	default:
		wfiLog.WithField("notification", n).Warn("Ignoring unknown notification type")
	}
	return nil
}

func (cr *Controller) Notify(msg *fes.Notification) error {
	wfiLog.WithFields(log.Fields{
		"notification": msg.EventType,
		"labels":       msg.Labels(),
	}).Debugf("Controller event: %v", msg.EventType)

	switch msg.EventType {
	case events.Invocation_INVOCATION_COMPLETED.String():
		fallthrough
	case events.Invocation_INVOCATION_CANCELED.String():
		fallthrough
	case events.Invocation_INVOCATION_FAILED.String():
		wfi, ok := msg.Payload.(*aggregates.WorkflowInvocation)
		if !ok {
			log.Warn("Event did not contain invocation payload", msg)
		}
		// TODO mark to clean up later instead
		cr.evalStates.Delete(wfi.ID())
		cr.evalStore.Delete(wfi.ID())
		log.Infof("Removed invocation %v from eval state", wfi.ID())
	case events.Task_TASK_FAILED.String():
		fallthrough
	case events.Task_TASK_SUCCEEDED.String():
		fallthrough
	case events.Invocation_INVOCATION_CREATED.String():
		wfi, ok := msg.Payload.(*aggregates.WorkflowInvocation)
		if !ok {
			panic(msg)
		}
		es := cr.evalStore.LoadOrStore(wfi.ID())
		cr.evalQueue.Push(es)
	default:
		wfiLog.Debugf("Controller ignored event type: %v", msg.EventType)
	}
	return nil
}

func (cr *Controller) Tick(tick uint64) error {
	// Short loop: invocations the controller is actively tracking
	var err error
	if tick%10 == 0 {
		log.Debug("Checking eval store for missing invocations")
		err = cr.checkEvalStore()
	}

	// Long loop: to check if there are any orphans
	if tick%50 == 0 {
		log.Debug("Checking model caches for missing invocations")
		err = cr.checkModelCaches()
	}

	return err
}

// <<<<<<< HEAD
// func (cr *Controller) checkEvalCaches() error {
// 	for id, state := range cr.evalCache.List() {
// 		last, ok := state.Last()
// 		if !ok {
// 			continue
// 		}

func (cr *Controller) checkEvalStore() error {
	for id, state := range cr.evalStore.List() {
		// TODO check if finished
		// Add all states that are somehow not in the evalQueue
		if cr.evalQueue.Get(id) == nil {
			log.Debugf("Adding missing invocation %v to the queue", id)
			cr.evalQueue.Push(state)
		}
	}
	return nil
}

// checkCaches iterates over the current caches submitting evaluation for invocation when needed
func (cr *Controller) checkModelCaches() error {
	// Short control loop
	entities := cr.invokeCache.List()
	for _, entity := range entities {
		// Ignore those that are in the evalStore; those will get picked up by checkEvalStore.
		if _, ok := cr.evalStore.Load(entity.Id); ok {
			continue
		}

		wi := aggregates.NewWorkflowInvocation(entity.Id)
		err := cr.invokeCache.Get(wi)
		if err != nil {
			wfiLog.Errorf("Failed to read '%v' from cache: %v.", wi.Aggregate(), err)
			continue
		}

		if !wi.Status.Finished() {
			controller.EvalRecovered.WithLabelValues(Name, "cache").Inc()
			es := cr.evalStore.LoadOrStore(wi.ID())
			cr.evalQueue.Push(es)
		}
	}
	return nil
}

func (cr *Controller) Evaluate(invocationID string) {
	start := time.Now()
	// Fetch and attempt to claim the evaluation
	evalState := cr.evalStore.LoadOrStore(invocationID)
	select {
	case <-evalState.Lock():
		defer evalState.Free()
	default:
		// TODO provide option to wait for a lock
		wfiLog.Debugf("Failed to obtain access to invocation %s", invocationID)
		controller.EvalJobs.WithLabelValues(Name, "duplicate").Inc()
		return
	}
	wfiLog.Debugf("Evaluating invocation %s", invocationID)

	// Fetch the workflow invocation for the provided invocation id
	wfi := aggregates.NewWorkflowInvocation(invocationID)
	err := cr.invokeCache.Get(wfi)
	// TODO move to rule
	if err != nil && wfi.WorkflowInvocation == nil {
		log.Errorf("controller failed to get invocation for invocation id '%s': %v", invocationID, err)
		controller.EvalJobs.WithLabelValues(Name, "error").Inc()
		return
	}
	// TODO move to rule
	if wfi.Status.Finished() {
		wfiLog.Debugf("No need to evaluate finished invocation %v", invocationID)
		controller.EvalJobs.WithLabelValues(Name, "error").Inc()
		return
	}

	// Fetch the workflow relevant to the invocation
	wf := aggregates.NewWorkflow(wfi.Spec.WorkflowId)
	err = cr.wfCache.Get(wf)
	// TODO move to rule
	if err != nil && wf.Workflow == nil {
		wfiLog.Errorf("controller failed to get workflow '%s' for invocation id '%s': %v", wfi.Spec.WorkflowId,
			invocationID, err)
		controller.EvalJobs.WithLabelValues(Name, "error").Inc()
		return
	}

	// Evaluate invocation
	record := controller.NewEvalRecord() // TODO implement rulepath + cause

	ec := NewEvalContext(evalState, wf.Workflow, wfi.WorkflowInvocation)

	action := cr.evalPolicy.Eval(ec)
	record.Action = action
	if action == nil {
		controller.EvalJobs.WithLabelValues(Name, "noop").Inc()
		return
	}

	// Execute action
	err = action.Apply()
	if err != nil {
		wfiLog.Errorf("Action '%T' failed: %v", action, err)
		record.Error = err
	}
	controller.EvalJobs.WithLabelValues(Name, "action").Inc()

	// Record this evaluation
	evalState.Record(record)

	controller.EvalDuration.WithLabelValues(Name, fmt.Sprintf("%T", action)).Observe(float64(time.Now().Sub(start)))
	if wfi.GetStatus().Finished() {
		t, _ := ptypes.Timestamp(wfi.GetMetadata().GetCreatedAt())
		invocationDuration.Observe(float64(time.Now().Sub(t)))
	}
	invocationStatus.WithLabelValues(wfi.GetStatus().GetStatus().String()).Inc()
}

func (cr *Controller) Close() error {
	if invokePub, ok := cr.invokeCache.(pubsub.Publisher); ok {
		err := invokePub.Unsubscribe(cr.fesSub)
		if err != nil {
			wfiLog.Errorf("Failed to unsubscribe from invocation cache: %v", err)
		} else {
			wfiLog.Info("Unsubscribed from invocation cache")
		}
	}

	cr.cancelFn()
	return nil
}

func (cr *Controller) createFailAction(invocationID string, err error) controller.Action {
	return &ActionFail{
		API:          cr.invocationAPI,
		InvocationID: invocationID,
		Err:          err,
	}
}

func defaultPolicy(ctr *Controller) controller.Rule {
	return &controller.RuleEvalUntilAction{
		Rules: []controller.Rule{
			&controller.RuleTimedOut{
				OnTimedOut: &ActionFail{
					API: ctr.invocationAPI,
					Err: errors.New("timed out"),
				},
				Timeout: time.Duration(10) * time.Minute,
			},
			&controller.RuleExceededErrorCount{
				OnExceeded: &ActionFail{
					API: ctr.invocationAPI,
				},
				MaxErrorCount: 0,
			},
			&RuleHasCompleted{},
			&RuleCheckIfCompleted{
				InvocationAPI: ctr.invocationAPI,
			},
			&RuleWorkflowIsReady{},
			&RuleSchedule{
				Scheduler:     ctr.scheduler,
				InvocationAPI: ctr.invocationAPI,
				FunctionAPI:   ctr.taskAPI,
				StateStore:    ctr.evalStates,
			},
		},
	}
}

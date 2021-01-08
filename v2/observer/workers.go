package main

import (
	"context"
	"fmt"
	"time"

	"github.com/brigadecore/brigade/sdk/v2/core"
	myk8s "github.com/brigadecore/brigade/v2/internal/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func (o *observer) syncWorkerPods(ctx context.Context) {
	workerPodsInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = myk8s.WorkerPodsSelector()
				return o.kubeClient.CoreV1().Pods("").List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = myk8s.WorkerPodsSelector()
				return o.kubeClient.CoreV1().Pods("").Watch(ctx, options)
			},
		},
		&corev1.Pod{},
		0,
		cache.Indexers{},
	)
	workerPodsInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: o.syncWorkerPodFn,
			UpdateFunc: func(_, newObj interface{}) {
				o.syncWorkerPodFn(newObj)
			},
			DeleteFunc: o.syncDeletedPodFn,
		},
	)
	workerPodsInformer.Run(ctx.Done())
}

func (o *observer) syncWorkerPod(obj interface{}) {
	pod := obj.(*corev1.Pod)

	// Worker pods are only deleted after we're FULLY done with them. So if the
	// DeletionTimestamp is set, there's nothing for us to do because the Worker
	// must already be in a terminal state.
	if pod.DeletionTimestamp != nil {
		return
	}

	status := core.WorkerStatus{}
	switch pod.Status.Phase {
	case corev1.PodPending:
		// For Brigade's purposes, this counts as running
		status.Phase = core.WorkerPhaseRunning
		// Unless... when an image pull backoff occurs, the pod still shows as
		// pending. We account for that here and treat it as a failure.
		//
		// TODO: Are there other conditions we need to watch out for?
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil &&
				(containerStatus.State.Waiting.Reason == "ImagePullBackOff" ||
					containerStatus.State.Waiting.Reason == "ErrImagePull") {
				status.Phase = core.WorkerPhaseFailed
				break
			}
		}
	case corev1.PodRunning:
		status.Phase = core.WorkerPhaseRunning
	case corev1.PodSucceeded:
		status.Phase = core.WorkerPhaseSucceeded
	case corev1.PodFailed:
		status.Phase = core.WorkerPhaseFailed
	case corev1.PodUnknown:
		status.Phase = core.WorkerPhaseUnknown
	}

	if pod.Status.StartTime != nil {
		status.Started = &pod.Status.StartTime.Time
	}
	// Pods don't really have an end time. We grab the end time of container[0]
	// because that's what we really care about.
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == pod.Spec.Containers[0].Name {
			if containerStatus.State.Terminated != nil {
				status.Ended =
					&pod.Status.ContainerStatuses[0].State.Terminated.FinishedAt.Time
			}
			break
		}
	}

	// Use the API to update Worker status
	eventID := pod.Labels[myk8s.LabelEvent]
	ctx, cancel := context.WithTimeout(context.Background(), apiRequestTimeout)
	defer cancel()
	if err := o.updateWorkerStatusFn(
		ctx,
		eventID,
		status,
	); err != nil {
		o.errFn(
			fmt.Sprintf(
				"error updating status for event %q worker: %s",
				eventID,
				err,
			),
		)
	}

	if status.Phase == core.WorkerPhaseSucceeded ||
		status.Phase == core.WorkerPhaseFailed {
		go o.deleteWorkerResourcesFn(pod.Namespace, pod.Name, eventID)
	}
}

// deleteWorkerResources deletes a Worker pod after a 60 second delay. The delay
// is to ensure any log aggregators have a chance to get all logs from the
// completed pod before it is torpedoed.
func (o *observer) deleteWorkerResources(namespace, podName, eventID string) {
	namespacedWorkerPodName := namespacedPodName(namespace, podName)

	o.syncMu.Lock()
	if _, alreadyDeleting :=
		o.deletingPodsSet[namespacedWorkerPodName]; alreadyDeleting {
		o.syncMu.Unlock()
		return
	}
	o.deletingPodsSet[namespacedWorkerPodName] = struct{}{}
	o.syncMu.Unlock()

	<-time.After(o.config.delayBeforeCleanup)

	ctx, cancel := context.WithTimeout(context.Background(), apiRequestTimeout)
	defer cancel()
	if err := o.cleanupWorkerFn(ctx, eventID); err != nil {
		o.errFn(
			fmt.Sprintf(
				"error cleaning up after worker for event %q: %s",
				eventID,
				err,
			),
		)
	}
}
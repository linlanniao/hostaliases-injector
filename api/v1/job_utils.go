package v1

import (
	"context"
	"errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

func (_ *JobMutate) parseAnnotation(job *batchv1.Job) []ReCreateReason {
	comparisonTypes := make([]ReCreateReason, 0)

	if len(job.Annotations) == 0 {
		return comparisonTypes
	}

	keys, ok := job.Annotations[AnnotationComparisonKey]
	if !ok {
		return comparisonTypes
	}

	m := map[ReCreateReason]struct{}{
		ReasonImageChange:   {},
		ReasonEnvChange:     {},
		ReasonEnvFromChange: {},
	}

	for _, key := range strings.Split(keys, ",") {
		k := ReCreateReason(key)
		if _, ok := m[k]; ok {
			comparisonTypes = append(comparisonTypes, k)
		}
	}

	return comparisonTypes
}

func (jm *JobMutate) GetJob(ctx context.Context, name, namespace string) (*batchv1.Job, error) {
	obj := &batchv1.Job{}
	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}
	if name == "" {
		return obj, errors.New("resource name is required")
	}
	objKey := types.NamespacedName{Namespace: namespace, Name: name}
	err := jm.Client.Get(ctx, objKey, obj)
	return obj, err
}

func (jm *JobMutate) CreateJob(ctx context.Context, job *batchv1.Job) error {
	job.Status = batchv1.JobStatus{}
	if job.Annotations != nil {
		delete(job.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}
	job.UID = ""
	job.ResourceVersion = ""
	job.Spec.Selector = nil
	if job.Spec.Template.Labels != nil {
		delete(job.Spec.Template.Labels, "controller-uid")
	}

	//return jm.Client.Create(ctx, job)
	return retry.OnError(retry.DefaultRetry, func(err error) bool {
		return true
	}, func() error {
		return jm.Client.Create(ctx, job)
	})
}

func (jm *JobMutate) DeleteJob(ctx context.Context, name, namespace string) error {

	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}
	if name == "" {
		return errors.New("resource name is required")
	}
	obj := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	policy := metav1.DeletePropagationBackground
	//policy := metav1.DeletePropagationForeground
	deleteOpts := &client.DeleteOptions{
		GracePeriodSeconds: utilpointer.Int64(0),
		Preconditions:      nil,
		PropagationPolicy:  &policy,
		Raw:                nil,
		DryRun:             nil,
	}

	done := make(chan error)

	go func() {
		for {
			err1 := retry.OnError(retry.DefaultRetry, func(err error) bool {
				return true
			}, func() error {
				return jm.Client.Delete(ctx, obj, deleteOpts)
			})
			time.Sleep(time.Millisecond * 500)
			if _, err2 := jm.GetJob(ctx, name, namespace); err2 != nil {
				done <- err1
			}
		}
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(time.Millisecond * 8000):
		return errors.New("timed out")
	}
}

func (jm *JobMutate) CompareJob(left, right *batchv1.Job) (isSame bool) {
	reasons := jm.parseAnnotation(left)

	lContainers := left.Spec.Template.Spec.Containers
	rContainers := right.Spec.Template.Spec.Containers

	m := make(map[ReCreateReason]struct{})
	for _, c := range reasons {
		m[c] = struct{}{}
	}

	if len(lContainers) != len(rContainers) {
		return false
	}

	for i := range lContainers {
		leftContainer := lContainers[i]
		rightContainer := rContainers[i]

		if _, ok := m[ReasonImageChange]; ok {
			if !jm.CompareContainerImage(leftContainer, rightContainer) {
				return false
			}
		}

		if _, ok := m[ReasonEnvChange]; ok {
			if !jm.CompareContainerEnv(leftContainer, rightContainer) {
				return false
			}
		}

		if _, ok := m[ReasonEnvFromChange]; ok {
			if !jm.CompareContainerEnvFrom(leftContainer, rightContainer) {
				return false
			}
		}
	}

	// finally
	return true
}

func (_ *JobMutate) CompareContainerImage(left, right corev1.Container) (isSame bool) {
	return left.Image == right.Image
}

func (_ *JobMutate) CompareContainerEnv(left, right corev1.Container) (isSame bool) {
	compareFunc := func(left, right corev1.EnvVar) bool {
		return left.String() == right.String()

	}
	if left.Env == nil && right.Env == nil {
		return true
	}
	if len(left.Env) != len(right.Env) {
		return false
	}

	for i := range left.Env {
		if !compareFunc(left.Env[i], right.Env[i]) {
			return false
		}
	}

	return true
}

func (_ *JobMutate) CompareContainerEnvFrom(left, right corev1.Container) (isSame bool) {
	compareFunc := func(left, right corev1.EnvFromSource) bool {
		return left.String() == right.String()
	}

	if left.EnvFrom == nil && right.EnvFrom == nil {
		return true
	}
	if len(left.EnvFrom) != len(right.EnvFrom) {
		return false
	}

	for i := range left.EnvFrom {
		if !compareFunc(left.EnvFrom[i], right.EnvFrom[i]) {
			return false
		}
	}

	return true
}

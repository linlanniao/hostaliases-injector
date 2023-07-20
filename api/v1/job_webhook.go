package v1

import (
	"context"
	"errors"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

//+kubebuilder:webhook:path=/mutate-batch-coreV1-job,mutating=true,failurePolicy=ignore,sideEffects=None,groups=batch,resources=jobs,verbs=create;update,versions=coreV1,name=mjob.kb.io,admissionReviewVersions=coreV1
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// log is for logging in this package.
var logger = logf.Log.WithName("job-mutator")

type JobMutate struct {
	Client  client.Client
	decoder *admission.Decoder
}

func NewJobMutate(client client.Client) admission.Handler {
	decoder := admission.NewDecoder(runtime.NewScheme())
	return &JobMutate{
		Client:  client,
		decoder: decoder,
	}
}

const (
	AnnotationKey = "job-mutator.sre.rootcloud.info/comparison-content"
)

type ComparisonType string

var (
	TypeComparisonImage   ComparisonType = "image"
	TypeComparisonEnv     ComparisonType = "env"
	TypeComparisonEnvFrom ComparisonType = "envFrom"
)

func (_ *JobMutate) parseAnnotation(job *batchV1.Job) []ComparisonType {
	comparisonTypes := make([]ComparisonType, 0)

	if job.Annotations == nil {
		return comparisonTypes
	}

	keys, ok := job.Annotations[AnnotationKey]
	if !ok {
		return comparisonTypes
	}

	m := map[ComparisonType]struct{}{
		TypeComparisonImage:   {},
		TypeComparisonEnv:     {},
		TypeComparisonEnvFrom: {},
	}

	for _, key := range strings.Split(keys, ",") {
		k := ComparisonType(key)
		if _, ok := m[k]; ok {
			comparisonTypes = append(comparisonTypes, k)
		}
	}

	return comparisonTypes
}

func (jm *JobMutate) GetJob(ctx context.Context, name, namespace string) (*batchV1.Job, error) {
	obj := &batchV1.Job{}
	if namespace == "" {
		namespace = coreV1.NamespaceDefault
	}
	if name == "" {
		return obj, errors.New("resource name is required")
	}
	objKey := types.NamespacedName{Namespace: namespace, Name: name}
	err := jm.Client.Get(ctx, objKey, obj)
	return obj, err
}

func (jm *JobMutate) DeleteJob(ctx context.Context, job *batchV1.Job) error {
	return jm.Client.Delete(ctx, job)
}

func (jm *JobMutate) Handle(ctx context.Context, req admission.Request) admission.Response {
	// TODO
	newJob := new(batchV1.Job)
	err := jm.decoder.Decode(req, newJob)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	oldJob, err := jm.GetJob(ctx, newJob.Name, newJob.Namespace)
	if err != nil {
		return admission.Allowed("oldJob not found, skipping.")
	}

	ComparisonTypes := jm.parseAnnotation(newJob)

	if len(ComparisonTypes) == 0 {
		return admission.Allowed("noting to compare, skipping.")
	}

	isSame := jm.CompareTemplate(newJob.Spec.Template.Spec, oldJob.Spec.Template.Spec, ComparisonTypes)
	if !isSame {
		logger.Info("comparing failed,delete older Job", oldJob.Namespace, oldJob.Name)
		err := jm.DeleteJob(ctx, oldJob)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}

	logger.Info("comparing passed, replace job.spec")
	newJob.Spec = oldJob.Spec
	logger.Info("replace new job.spec")

	resp, err := json.Marshal(newJob)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, resp)
}

func (_ *JobMutate) CompareContainerImage(left, right coreV1.Container) (isSame bool) {
	return left.Image == right.Image
}

func (_ *JobMutate) CompareContainerEnv(left, right coreV1.Container) (isSame bool) {
	compareFunc := func(left, right coreV1.EnvVar) bool {
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

func (_ *JobMutate) CompareContainerEnvFrom(left, right coreV1.Container) (isSame bool) {
	compareFunc := func(left, right coreV1.EnvFromSource) bool {
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

func (jm *JobMutate) CompareTemplate(left, right coreV1.PodSpec, comparisonContent []ComparisonType) (isSame bool) {
	m := make(map[ComparisonType]struct{})
	for _, c := range comparisonContent {
		m[c] = struct{}{}
	}

	if len(left.Containers) != len(right.Containers) {
		return false
	}

	for i := range left.Containers {
		leftContainer := left.Containers[i]
		rightContainer := right.Containers[i]

		if _, ok := m[TypeComparisonImage]; ok {
			if jm.CompareContainerImage(leftContainer, rightContainer) {
				return false
			}
		}

		if _, ok := m[TypeComparisonEnv]; ok {
			if jm.CompareContainerEnv(leftContainer, rightContainer) {
				return false
			}
		}

		if _, ok := m[TypeComparisonEnvFrom]; ok {
			if jm.CompareContainerEnvFrom(leftContainer, rightContainer) {
				return false
			}
		}
	}

	// finally
	return true
}

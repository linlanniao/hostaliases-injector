package v1

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"time"
)

//+kubebuilder:webhook:path=/mutate-batch-v1-job,mutating=true,failurePolicy=ignore,sideEffects=None,groups=batch,resources=jobs,verbs=update,versions=v1,name=mjob.kb.io,admissionReviewVersions=v1
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
	AnnotationComparisonKey = "job-mutator.sre.rootcloud.info/comparison-content"
	AnnotationProcessingKey = "job-mutator.sre.rootcloud.info/processing"
)

type ComparisonType string

var (
	TypeComparisonImage   ComparisonType = "image"
	TypeComparisonEnv     ComparisonType = "env"
	TypeComparisonEnvFrom ComparisonType = "envFrom"
)

func (jm *JobMutate) Handle(ctx context.Context, req admission.Request) admission.Response {
	newJob := new(batchv1.Job)
	err := jm.decoder.Decode(req, newJob)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if len(newJob.Annotations) == 0 {
		logger.Info(
			"annotations is empty",
			"action", "skip",
			"namespace", newJob.Namespace,
			"name", newJob.Name,
		)
		return admission.Allowed("")
	}
	if _, ok := newJob.Annotations[AnnotationComparisonKey]; !ok {
		logger.Info(
			"uncontrolled target",
			"action", "skip",
			"namespace", newJob.Namespace,
			"name", newJob.Name,
		)
		return admission.Allowed("")
	}

	oldJob, err := jm.GetJob(ctx, newJob.Name, newJob.Namespace)
	if err != nil {
		logger.Info(
			"oldJob not found",
			"action", "skip",
			"namespace", newJob.Namespace,
			"name", newJob.Name,
		)
		return admission.Allowed("")
	}

	if jm.isProcessing(newJob) {
		delete(newJob.Annotations, AnnotationProcessingKey)
		resp, err := json.Marshal(newJob)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		return admission.PatchResponseFromRaw(req.Object.Raw, resp)
	}

	ComparisonTypes := jm.parseAnnotation(newJob)

	if len(ComparisonTypes) == 0 {
		logger.Info(
			"no comparison types found",
			"action", "skip",
			"namespace", newJob.Namespace,
			"name", newJob.Name,
		)
		return admission.Allowed("")
	}

	isSame := jm.CompareJob(newJob, oldJob)
	if !isSame {

		if err := jm.DeleteJob(ctx, oldJob.Name, oldJob.Namespace); err != nil {
			logger.Error(err,
				"failed to delete job",
				"action", "DeleteJob",
				"namespace", oldJob.Namespace,
				"name", oldJob.Name,
			)
		}

		if len(newJob.Annotations) == 0 {
			newJob.Annotations = make(map[string]string)
		}
		now := time.Now()
		newJob.Annotations[AnnotationProcessingKey] = now.UTC().Format("2006-01-02T15:04:05Z")

		if err := jm.CreateJob(ctx, newJob); err != nil {
			logger.Error(err,
				"failed to create job",
				"action", "CreateJob",
				"namespace", newJob.Namespace,
				"name", newJob.Name,
			)
		}
		logger.Info(
			"difference content",
			"comparisonTypes", newJob.Annotations[AnnotationComparisonKey],
			"action", "delete and recreate",
			"namespace", newJob.Namespace,
			"name", newJob.Name,
		)

		resp, err := json.Marshal(newJob)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		return admission.PatchResponseFromRaw(req.Object.Raw, resp)
	}

	newJob.Spec = oldJob.Spec
	logger.Info(
		"same content",
		"comparisonTypes", newJob.Annotations[AnnotationComparisonKey],
		"action", "update metadata only",
		"namespace", newJob.Namespace,
		"name", newJob.Name,
	)

	resp, err := json.Marshal(newJob)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, resp)
}

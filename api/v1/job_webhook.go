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
)

// log is for logging in this package.
var joblog = logf.Log.WithName("job-mutator")

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-batch-v1-job,mutating=true,failurePolicy=ignore,sideEffects=None,groups=batch,resources=jobs,verbs=create;update,versions=v1,name=mjob.kb.io,admissionReviewVersions=v1
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

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

func (v *JobMutate) Handle(ctx context.Context, req admission.Request) admission.Response {
	// TODO
	job := &batchv1.Job{}
	err := v.decoder.Decode(req, job)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if job.Labels == nil {
		job.Labels = make(map[string]string)
	}

	if _, ok := job.Labels["testa"]; ok {
		delete(job.Labels, "testa")
	}
	//job.Labels["testa"] = "testbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	resp, err := json.Marshal(job)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	for k, v := range job.GetLabels() {
		joblog.Info("labeled", k, v)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, resp)
}

//func (v *JobMutate) InjectDecoder(d *admission.Decoder) error {
//	v.decoder = d
//	return nil
//}

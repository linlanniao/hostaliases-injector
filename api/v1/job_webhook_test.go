package v1

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func newJm() JobMutate {
	cfg, err := config.GetConfig()
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return JobMutate{
		Client:  c,
		decoder: admission.NewDecoder(runtime.NewScheme()),
	}
}

func TestGetJob(t *testing.T) {
	jm := newJm()
	job, err := jm.GetJob(context.TODO(), "devops-xxl-job-patch-job-latest", "")
	if err != nil {
		t.Log(err.Error())
	}
	t.Log(job.String())
}

func TestDeleteJob(t *testing.T) {
	jm := newJm()
	job, err := jm.GetJob(context.TODO(), "devops-xxl-job-patch-job-latest", "")
	if err != nil {
		t.Log(err.Error())
	}
	t.Log(job.String())
	t.Log("--------------------------------")
	err2 := jm.DeleteJob(context.TODO(), job.Name, job.Namespace)
	if err2 != nil {
		t.Log(err.Error())
	}
}

func TestReplaceJob(t *testing.T) {
	jm := newJm()
	job, err := jm.GetJob(context.TODO(), "devops-xxl-job-patch-job-latest", "")
	if err != nil {
		t.Log(err.Error())
	}
	t.Log(job.String())
	t.Log("--------------------------------")
	err2 := jm.DeleteJob(context.TODO(), job.Name, job.Namespace)
	if err2 != nil {
		t.Log(err2.Error())
	}
	//time.Sleep(time.Millisecond * 1500)
	err3 := jm.CreateJob(context.TODO(), job)
	if err3 != nil {
		t.Log(err3.Error())
	}
}

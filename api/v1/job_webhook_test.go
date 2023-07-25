package v1

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/admission/v1"
	batchv1 "k8s.io/api/batch/v1"
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
	job, err := jm.GetJob(context.TODO(), "test-job-latest", "")
	if err != nil {
		t.Log(err.Error())
	}
	t.Log(job.String())
}

func TestDeleteJob(t *testing.T) {
	jm := newJm()
	job, err := jm.GetJob(context.TODO(), "test-job-latest", "")
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
	job, err := jm.GetJob(context.TODO(), "test-job-latest", "")
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

func raw2Request(raw string) admission.Request {
	return admission.Request{
		AdmissionRequest: v1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw:    []byte(raw),
				Object: nil,
			},
		},
	}
}

func TestCompareImage(t *testing.T) {
	raw1 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-mutator.sre.rootcloud.info/comparison-content: image
  labels:
    app.oam.dev/revision: test-job-v1
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        image: dockerhub.tencentcloudcr.com/library/nginx:alpine
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	raw2 := `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    job-mutator.sre.rootcloud.info/comparison-content: image
  labels:
    app.oam.dev/revision: test-job-v2
  name: test-job-latest
  namespace: default
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  suspend: false
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test-job
        app.oam.dev/component: test-job
        job-name: test-job-latest
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - |
          echo default
          sleep 10
        env:
        - name: k1
          value: v1
        - name: k2
          value: v2
        image: dockerhub.tencentcloudcr.com/library/nginx:alpine
        #image: dockerhub.tencentcloudcr.com/library/nginx:stable
        imagePullPolicy: Always
        name: test-job
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 900
`
	leftReq := raw2Request(raw1)
	rightReq := raw2Request(raw2)
	jm := newJm()
	lJob := &batchv1.Job{}
	rJob := &batchv1.Job{}
	_ = jm.decoder.Decode(leftReq, lJob)
	_ = jm.decoder.Decode(rightReq, rJob)
	isSame := jm.CompareJob(lJob, rJob)
	assert.True(t, isSame)
}

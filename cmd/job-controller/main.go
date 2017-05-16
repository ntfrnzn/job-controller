package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	batchV1 "k8s.io/client-go/pkg/apis/batch/v1"
	"k8s.io/client-go/rest"
)

// JobController is the job controller
type JobController struct {
	clientset   *kubernetes.Clientset
	namespace   string
	jobTemplate string
}

func newInClusterJobController() *JobController {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespace := "default"

	configmaps := clientset.Core().ConfigMaps(namespace)
	cm, err := configmaps.Get("job-template")
	if err != nil {
		panic(err.Error())
	}
	template := cm.Data["example-job.json"]

	fmt.Printf("Got initial template data of: %s\n", template)

	return &JobController{
		clientset:   clientset,
		namespace:   namespace,
		jobTemplate: template,
	}
}

// see
//   https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/batch/v1/job.go
//   https://github.com/kubernetes/client-go/blob/master/pkg/apis/batch/v1/types.go
func (jc *JobController) inspectJobs() {
	jobs, err := jc.clientset.BatchV1().Jobs(jc.namespace).List(v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for _, job := range jobs.Items {
		if job.Status.Active == 0 && job.Status.Succeeded > 0 {

			fmt.Printf("Job %s has completed, deleting it (and trashing its pods/logs) now \n", job.Name)

			//deletePods := true
			deleteOptions := v1.DeleteOptions{} // v1.DeleteOptions{OrphanDependents: &deletePods}

			err := jc.clientset.BatchV1().Jobs(jc.namespace).Delete(job.Name, &deleteOptions)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (jc *JobController) countJobs() int {
	jobs, err := jc.clientset.BatchV1().Jobs("").List(v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	return len(jobs.Items)
}

type jobTemplater struct {
	JobToken string
}

func (jc *JobController) getSimpleJob(token string) *batchV1.Job {

	var serializedJob bytes.Buffer
	values := jobTemplater{
		JobToken: token,
	}
	t, err := template.New(token).Parse(jc.jobTemplate)
	if err != nil {
		panic(err)
	}

	err = t.Execute(&serializedJob, values)
	if err != nil {
		panic(err)
	}

	//fmt.Printf("Got job template of: %s\n", serializedJob.String())

	var job batchV1.Job
	err = json.Unmarshal(serializedJob.Bytes(), &job)
	if err != nil {
		panic(err.Error())
	}
	return &job
}

func (jc *JobController) submitSimpleJob(token string) {
	job := jc.getSimpleJob(token)
	submitted, err := jc.clientset.BatchV1().Jobs(jc.namespace).Create(job)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Submitted job %s\n", submitted.Name)
}

func main() {

	jc := newInClusterJobController()

	for {
		t := time.Now()
		jc.submitSimpleJob(t.Format(time.RFC3339))
		fmt.Printf("There are %d jobs in the cluster\n", jc.countJobs())
		jc.inspectJobs()
		time.Sleep(10 * time.Second)
	}
}

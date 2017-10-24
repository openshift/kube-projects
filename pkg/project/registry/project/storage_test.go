package project

import (
	"strings"
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openshift/kube-projects/pkg/apis/project"
)

// mockLister returns the namespaces in the list
type mockLister struct {
	namespaceList *v1.NamespaceList
}

func (ml *mockLister) List(user user.Info) (*v1.NamespaceList, error) {
	return ml.namespaceList, nil
}

func TestListProjects(t *testing.T) {
	namespaceList := v1.NamespaceList{
		Items: []v1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
			},
		},
	}
	mockClient := fake.NewSimpleClientset(&namespaceList)
	storage := REST{
		client: mockClient.Core().Namespaces(),
		lister: &mockLister{&namespaceList},
	}
	user := &user.DefaultInfo{
		Name:   "test-user",
		UID:    "test-uid",
		Groups: []string{"test-groups"},
	}
	ctx := request.WithUser(request.NewContext(), user)
	response, err := storage.List(ctx, nil)
	if err != nil {
		t.Errorf("%#v should be nil.", err)
	}
	projects := response.(*project.ProjectList)
	if len(projects.Items) != 1 {
		t.Errorf("%#v projects.Items should have len 1.", projects.Items)
	}
	responseProject := projects.Items[0]
	if e, r := responseProject.Name, "foo"; e != r {
		t.Errorf("%#v != %#v.", e, r)
	}
}

func TestCreateProjectBadObject(t *testing.T) {
	storage := REST{}

	obj, err := storage.Create(request.NewContext(), &project.ProjectList{}, false)
	if obj != nil {
		t.Errorf("Expected nil, got %v", obj)
	}
	if strings.Index(err.Error(), "not a project:") == -1 {
		t.Errorf("Expected 'not an project' error, got %v", err)
	}
}

func TestCreateInvalidProject(t *testing.T) {
	mockClient := &fake.Clientset{}
	storage := NewREST(mockClient.Core().Namespaces(), &mockLister{}, nil, nil)
	_, err := storage.Create(request.NewContext(), &project.Project{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"openshift.io/display-name": "h\t\ni"},
		},
	}, false)
	if !errors.IsInvalid(err) {
		t.Errorf("Expected 'invalid' error, got %v", err)
	}
}

func TestCreateProjectOK(t *testing.T) {
	mockClient := &fake.Clientset{}
	storage := NewREST(mockClient.Core().Namespaces(), &mockLister{}, nil, nil)
	_, err := storage.Create(request.NewContext(), &project.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
	}, false)
	if err != nil {
		t.Errorf("Unexpected non-nil error: %#v", err)
	}
	if len(mockClient.Actions()) != 1 {
		t.Errorf("Expected client action for create")
	}
	if !mockClient.Actions()[0].Matches("create", "namespaces") {
		t.Errorf("Expected call to create-namespace")
	}
}

func TestGetProjectOK(t *testing.T) {
	mockClient := fake.NewSimpleClientset(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	storage := NewREST(mockClient.Core().Namespaces(), &mockLister{}, nil, nil)
	projectObj, err := storage.Get(request.NewContext(), "foo", nil)
	if projectObj == nil {
		t.Error("Unexpected nil project")
	}
	if err != nil {
		t.Errorf("Unexpected non-nil error: %v", err)
	}
	if projectObj.(*project.Project).Name != "foo" {
		t.Errorf("Unexpected project: %#v", projectObj)
	}
}

func TestDeleteProject(t *testing.T) {
	mockClient := &fake.Clientset{}
	storage := REST{
		client: mockClient.Core().Namespaces(),
	}
	obj, err := storage.Delete(request.NewContext(), "foo")
	if obj == nil {
		t.Error("Unexpected nil obj")
	}
	if err != nil {
		t.Errorf("Unexpected non-nil error: %#v", err)
	}
	status, ok := obj.(*metav1.Status)
	if !ok {
		t.Errorf("Expected status type, got: %#v", obj)
	}
	if status.Status != metav1.StatusSuccess {
		t.Errorf("Expected status=success, got: %#v", status)
	}
	if len(mockClient.Actions()) != 1 {
		t.Errorf("Expected client action for delete")
	}
	if !mockClient.Actions()[0].Matches("delete", "namespaces") {
		t.Errorf("Expected call to delete-namespace")
	}
}

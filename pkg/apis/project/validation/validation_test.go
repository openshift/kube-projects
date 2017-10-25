package validation

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openshift/kube-projects/pkg/apis/project"
)

func TestValidateProject(t *testing.T) {
	testCases := []struct {
		name    string
		project project.Project
		numErrs int
	}{
		{
			name: "missing id",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						project.ProjectDescription: "This is a description",
						project.ProjectDisplayName: "hi",
					},
				},
			},
			// Should fail because the ID is missing.
			numErrs: 1,
		},
		{
			name: "invalid id",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "141-.124.$",
					Annotations: map[string]string{
						project.ProjectDescription: "This is a description",
						project.ProjectDisplayName: "hi",
					},
				},
			},
			// Should fail because the ID is invalid.
			numErrs: 1,
		},
		{
			name: "invalid id uppercase",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "AA",
				},
			},
			numErrs: 1,
		},
		{
			name: "valid id leading number",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "11",
				},
			},
			numErrs: 0,
		},
		{
			name: "invalid id for create (< 2 characters)",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "h",
				},
			},
			numErrs: 1,
		},
		{
			name: "valid id for create (2+ characters)",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hi",
				},
			},
			numErrs: 0,
		},
		{
			name: "invalid id internal dots",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "1.a.1",
				},
			},
			numErrs: 1,
		},
		{
			name: "has namespace",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo",
					Annotations: map[string]string{
						project.ProjectDescription: "This is a description",
						project.ProjectDisplayName: "hi",
					},
				},
			},
			// Should fail because the namespace is supplied.
			numErrs: 1,
		},
		{
			name: "invalid display name",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "",
					Annotations: map[string]string{
						project.ProjectDescription: "This is a description",
						project.ProjectDisplayName: "h\t\ni",
					},
				},
			},
			// Should fail because the display name has \t \n
			numErrs: 1,
		},
		{
			name: "valid node selector",
			project: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "",
					Annotations: map[string]string{
						project.ProjectNodeSelector: "infra=true, env = test",
					},
				},
			},
			numErrs: 0,
		},
		// {
		// 	name: "invalid node selector",
		// 	project: project.Project{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "foo",
		// 			Namespace: "",
		// 			Annotations: map[string]string{
		// 				project.ProjectNodeSelector: "infra, env = $test",
		// 			},
		// 		},
		// 	},
		// 	// Should fail because infra and $test doesn't satisfy the format
		// 	numErrs: 1,
		// },
	}

	for _, tc := range testCases {
		errs := ValidateProject(&tc.project)
		if len(errs) != tc.numErrs {
			t.Errorf("Unexpected error list for case %q: %+v", tc.name, errs)
		}
	}

	project := project.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
			Annotations: map[string]string{
				project.ProjectDescription: "This is a description",
				project.ProjectDisplayName: "hi",
			},
		},
	}
	errs := ValidateProject(&project)
	if len(errs) != 0 {
		t.Errorf("Unexpected non-zero error list: %#v", errs)
	}
}

func TestValidateProjectUpdate(t *testing.T) {
	// Ensure we can update projects with short names, to make sure we can
	// proxy updates to namespaces created outside project validation
	projectObj := &project.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "project-name",
			ResourceVersion: "1",
			Annotations: map[string]string{
				project.ProjectDescription:  "This is a description",
				project.ProjectDisplayName:  "display name",
				project.ProjectNodeSelector: "infra=true, env = test",
			},
			Labels: map[string]string{"label-name": "value"},
		},
	}
	updateDisplayname := &project.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "project-name",
			ResourceVersion: "1",
			Annotations: map[string]string{
				project.ProjectDescription:  "This is a description",
				project.ProjectDisplayName:  "display name change",
				project.ProjectNodeSelector: "infra=true, env = test",
			},
			Labels: map[string]string{"label-name": "value"},
		},
	}

	errs := ValidateProjectUpdate(updateDisplayname, projectObj)
	if len(errs) > 0 {
		t.Fatalf("Expected no errors, got %v", errs)
	}

	errorCases := map[string]struct {
		A project.Project
		T field.ErrorType
		F string
	}{
		"change name": {
			A: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "different",
					ResourceVersion: "1",
					Annotations:     projectObj.Annotations,
					Labels:          projectObj.Labels,
				},
			},
			T: field.ErrorTypeInvalid,
			F: "metadata.name",
		},
		"invalid displayname": {
			A: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "project-name",
					ResourceVersion: "1",
					Annotations: map[string]string{
						project.ProjectDescription:  "This is a description",
						project.ProjectDisplayName:  "display name\n",
						project.ProjectNodeSelector: "infra=true, env = test",
					},
					Labels: projectObj.Labels,
				},
			},
			T: field.ErrorTypeInvalid,
			F: "metadata.annotations[" + project.ProjectDisplayName + "]",
		},
		"updating disallowed annotation": {
			A: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "project-name",
					ResourceVersion: "1",
					Annotations: map[string]string{
						project.ProjectDescription:  "This is a description",
						project.ProjectDisplayName:  "display name",
						project.ProjectNodeSelector: "infra=true, env = test2",
					},
					Labels: projectObj.Labels,
				},
			},
			T: field.ErrorTypeInvalid,
			F: "metadata.annotations[openshift.io/node-selector]",
		},
		"delete annotation": {
			A: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "project-name",
					ResourceVersion: "1",
					Annotations: map[string]string{
						project.ProjectDescription: "This is a description",
						project.ProjectDisplayName: "display name",
					},
					Labels: projectObj.Labels,
				},
			},
			T: field.ErrorTypeInvalid,
			F: "metadata.annotations[openshift.io/node-selector]",
		},
		"updating label": {
			A: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "project-name",
					ResourceVersion: "1",
					Annotations:     projectObj.Annotations,
					Labels:          map[string]string{"label-name": "diff"},
				},
			},
			T: field.ErrorTypeInvalid,
			F: "metadata.labels[label-name]",
		},
		"deleting label": {
			A: project.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "project-name",
					ResourceVersion: "1",
					Annotations:     projectObj.Annotations,
				},
			},
			T: field.ErrorTypeInvalid,
			F: "metadata.labels[label-name]",
		},
	}
	for k, v := range errorCases {
		errs := ValidateProjectUpdate(&v.A, projectObj)
		if len(errs) == 0 {
			t.Errorf("expected failure %s for %v", k, v.A)
			continue
		}
		for i := range errs {
			if errs[i].Type != v.T {
				t.Errorf("%s: expected errors to have type %s: %v", k, v.T, errs[i])
			}
			if errs[i].Field != v.F {
				t.Errorf("%s: expected errors to have field %s: %v", k, v.F, errs[i])
			}
		}
	}

}

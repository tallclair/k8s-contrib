package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/apis/core"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func main() {
	var kubeconfig *string
	home, err := os.UserHomeDir()
	errPanic(err)

	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	errPanic(err)

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	errPanic(err)

	escalatingRoles := map[rbacv1.RoleRef]bool{}

	clusterroles, err := clientset.RbacV1().ClusterRoles().List(metav1.ListOptions{})
	errPanic(err)
	for _, cr := range clusterroles.Items {
		if listEscalations(fmt.Sprintf("ClusterRole %s", cr.Name), cr.Rules) {
			ref := rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     cr.Name,
			}
			escalatingRoles[ref] = true
		}
	}

	sysroles, err := clientset.RbacV1().Roles(core.NamespaceSystem).List(metav1.ListOptions{})
	errPanic(err)
	for _, role := range sysroles.Items {
		if listEscalations(fmt.Sprintf("System Role %s/%s", role.Namespace, role.Name), role.Rules) {
			ref := rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     role.Name,
			}
			escalatingRoles[ref] = true
		}
	}
	fmt.Println()

	escalators := map[rbacv1.Subject]bool{}
	crbs, err := clientset.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{})
	errPanic(err)
	for _, crb := range crbs.Items {
		if escalatingRoles[crb.RoleRef] {
			for _, subj := range crb.Subjects {
				if subj.Kind == rbacv1.ServiceAccountKind {
					escalators[subj] = true
				}
			}
			printEscalatingRB(fmt.Sprintf("ClusterRoleBinding %s", crb.Name), crb.Subjects)
		}
	}
	sysrbs, err := clientset.RbacV1().RoleBindings(core.NamespaceSystem).List(metav1.ListOptions{})
	errPanic(err)
	for _, rb := range sysrbs.Items {
		if escalatingRoles[rb.RoleRef] {
			for _, subj := range rb.Subjects {
				if subj.Kind == rbacv1.ServiceAccountKind {
					escalators[subj] = true
				}
			}
			printEscalatingRB(fmt.Sprintf("RoleBinding %s/%s", rb.Namespace, rb.Name), rb.Subjects)
		}
	}
	fmt.Println()

	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	errPanic(err)
	for _, pod := range pods.Items {
		sa := rbacv1.Subject{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      pod.Spec.ServiceAccountName,
			Namespace: pod.Namespace,
		}
		if escalators[sa] {
			fmt.Printf("Pod %s/%s (%s) has escalating permissions\n", pod.Namespace, pod.Name, pod.Spec.ServiceAccountName)
		}
	}
}

func printEscalatingRB(descriptor string, subjects []rbacv1.Subject) {
	fmt.Printf("%s is escalating:\n", descriptor)
	for _, subject := range subjects {
		if subject.Kind == rbacv1.ServiceAccountKind {
			fmt.Printf("    %s/%s\n", subject.Namespace, subject.Name)
		}
	}
}

func listEscalations(descriptor string, rules []rbacv1.PolicyRule) bool {
	esc := getRuleEscalations(rules)
	if len(esc) == 0 {
		return false
	}

	fmt.Printf("%s is escalating:\n", descriptor)
	for res, verbs := range esc {
		fmt.Printf("    %s %s\n", strings.Join(verbs.List(), "/"), res)
	}
	return true
}

func getRuleEscalations(rules []rbacv1.PolicyRule) map[string]sets.String {
	escalations := map[string]sets.String{}
	for _, rule := range rules {
		addEscalations(escalations, rule)
	}
	return escalations
}

func addEscalations(escalations map[string]sets.String, rule rbacv1.PolicyRule) {
	qualifiedResources := []string{}
	for _, group := range rule.APIGroups {
		for _, res := range rule.Resources {
			qualified := group + "/" + res
			if group == "" {
				qualified = res
			}
			qualifiedResources = append(qualifiedResources, qualified)
		}
	}
	verbs := sets.NewString()
	for _, v := range rule.Verbs {
		switch v {
		case "get", "list", "watch":
			verbs.Insert("read")
		case "update", "patch":
			verbs.Insert("update")
		default:
			verbs.Insert(v)
		}
	}
	escalatingVerbs := sets.NewString("create", "update", "proxy")
	for _, res := range qualifiedResources {
		switch res {
		case "secrets":
			escalations[res] = escalations[res].Union(verbs)
		case
			"apps/daemonsets",
			"apps/deployments",
			"apps/replicasets",
			"apps/statefulsets",
			"authentication.k8s.io/tokenrequests",
			"batch/jobs",
			"configmaps",
			"extensions/daemonsets",
			"extensions/deployments",
			"extensions/replicasets",
			"nodes",
			"pods",
			"rbac.authorization.k8s.io/clusterrolebindings",
			"rbac.authorization.k8s.io/clusterroles",
			"rbac.authorization.k8s.io/rolebindings",
			"rbac.authorization.k8s.io/roles",
			"replicationcontrollers",
			"serviceaccount":
			if esc := verbs.Intersection(escalatingVerbs); len(esc) > 0 {
				escalations[res] = escalations[res].Union(esc)
			}
		}

		if idx := strings.LastIndex(res, "/"); idx != -1 {
			subres := res[idx+1:]
			switch subres {
			case "exec", "attach", "portforward", "proxy":
				escalations[res] = escalations[res].Union(verbs)
			}
		}
	}
}

func errPanic(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

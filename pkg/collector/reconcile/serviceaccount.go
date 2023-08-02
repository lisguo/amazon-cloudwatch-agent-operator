package reconcile

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aws/amazon-cloudwatch-agent-operator/apis/v1alpha1"
	"github.com/aws/amazon-cloudwatch-agent-operator/pkg/collector"
)

// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete

// ServiceAccounts reconciles the service account(s) required for the instance in the current context.
func ServiceAccounts(ctx context.Context, params Params) error {
	desired := desiredServiceAccounts(params)

	// first, handle the create/update parts
	if err := expectedServiceAccounts(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected service accounts: %w", err)
	}

	// then, delete the extra objects
	if err := deleteServiceAccounts(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the service accounts to be deleted: %w", err)
	}

	return nil
}

func desiredServiceAccounts(params Params) []corev1.ServiceAccount {
	desired := []corev1.ServiceAccount{}
	if params.Instance.Spec.Mode != v1alpha1.ModeSidecar && len(params.Instance.Spec.ServiceAccount) == 0 {
		desired = append(desired, collector.ServiceAccount(params.Instance))
	}
	return desired
}

func expectedServiceAccounts(ctx context.Context, params Params, expected []corev1.ServiceAccount) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &corev1.ServiceAccount{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if clientErr := params.Client.Create(ctx, &desired); clientErr != nil {
				return fmt.Errorf("failed to create: %w", clientErr)
			}
			params.Log.V(2).Info("created", "serviceaccount.name", desired.Name, "serviceaccount.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %w", err)
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "serviceaccount.name", desired.Name, "serviceaccount.namespace", desired.Namespace)
	}

	return nil
}

func deleteServiceAccounts(ctx context.Context, params Params, expected []corev1.ServiceAccount) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "amazon-cloudwatch-agent-operator",
		}),
	}
	list := &corev1.ServiceAccountList{}
	if err := params.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		existing := list.Items[i]
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
				break
			}
		}

		if del {
			if err := params.Client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
			params.Log.V(2).Info("deleted", "serviceaccount.name", existing.Name, "serviceaccount.namespace", existing.Namespace)
		}
	}

	return nil
}

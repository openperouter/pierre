package controller

import (
	"context"
	"fmt"
	"sort"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func nodeIndex(ctx context.Context, cli client.Client, node string) (int, error) {
	var nodes v1.NodeList
	if err := cli.List(ctx, &nodes); err != nil {
		return 0, fmt.Errorf("failed to get router pod for node %s: %v", node, err)
	}

	sort.Slice(nodes.Items, func(i, j int) bool {
		creationTimeI := nodes.Items[i].CreationTimestamp
		creationTimeJ := nodes.Items[j].CreationTimestamp
		if creationTimeI.Compare(creationTimeJ.Time) < 0 {
			return true
		}
		return false
	})
	return 0, nil
}

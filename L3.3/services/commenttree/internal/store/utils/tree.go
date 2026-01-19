package store

import (
	"commenttree/internal/dto"
	stErr "commenttree/internal/store/postgres/errors"
)

// BuildCommentTree разворачивает плоский список узлов в nested дерево.
// Важно: flat должен содержать rootID (корень).
func BuildCommentTree(rootID int64, flat []dto.CommentNode) (dto.CommentNode, error) {
	// id -> node
	nodes := make(map[int64]dto.CommentNode, len(flat))

	// parentID -> []childID
	childrenMap := make(map[int64][]int64)

	for _, n := range flat {

		n.Children = nil
		n.ChildrenCount = 0

		nodes[n.ID] = n

		if n.ParentID != nil {
			childrenMap[*n.ParentID] = append(childrenMap[*n.ParentID], n.ID)
		}
	}

	if _, ok := nodes[rootID]; !ok {
		return dto.CommentNode{}, stErr.ErrNotFound
	}

	// рекурсивно собираем children
	var build func(id int64) dto.CommentNode
	build = func(id int64) dto.CommentNode {
		n := nodes[id]

		childIDs := childrenMap[id]
		if len(childIDs) > 0 {
			n.Children = make([]dto.CommentNode, 0, len(childIDs))
			for _, cid := range childIDs {
				n.Children = append(n.Children, build(cid))
			}
		}

		n.ChildrenCount = len(n.Children)
		return n
	}

	return build(rootID), nil
}

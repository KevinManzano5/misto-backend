-- name: DeleteStoreCategoryAssignments :exec
DELETE FROM store_category_assignments
WHERE store_id = $1;

-- name: InsertStoreCategoryAssignment :exec
INSERT INTO store_category_assignments (store_id, category_id)
VALUES ($1, $2);

-- name: ListCategoriesForStore :many
SELECT sc.*
FROM store_categories sc
JOIN store_category_assignments a ON a.category_id = sc.id
WHERE a.store_id = $1
ORDER BY sc.name;

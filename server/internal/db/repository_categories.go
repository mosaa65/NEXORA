package db

import (
	"context"
	"fmt"
	"strings"
)

func (r *Repository) ListCategories(ctx context.Context) ([]CategorySummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			c.id,
			c.name_ar,
			c.name_en,
			c.slug,
			COUNT(DISTINCT mi.id) AS media_count,
			COUNT(vf.id) AS file_count
		FROM categories c
		LEFT JOIN media_items mi ON mi.category_id = c.id
		LEFT JOIN video_files vf ON vf.media_item_id = mi.id
		GROUP BY c.id
		ORDER BY c.name_en;
	`)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	categories := make([]CategorySummary, 0)
	for rows.Next() {
		var category CategorySummary
		var mediaCount, fileCount int64
		if err := rows.Scan(&category.ID, &category.NameAR, &category.NameEN, &category.Slug, &mediaCount, &fileCount); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		category.MediaCount = int(mediaCount)
		category.FileCount = int(fileCount)
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return categories, nil
}

func (r *Repository) CreateCategory(ctx context.Context, nameAR, nameEN, slug string) (*CategorySummary, error) {
	nameAR = strings.TrimSpace(nameAR)
	nameEN = strings.TrimSpace(nameEN)
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(nameEN, " ", "-"))
	}
	if nameEN == "" {
		nameEN = nameAR
	}
	if nameAR == "" {
		nameAR = nameEN
	}

	var newID int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO categories (name_ar, name_en, slug)
		VALUES ($1, $2, $3)
		ON CONFLICT (slug) DO UPDATE SET name_ar = EXCLUDED.name_ar, name_en = EXCLUDED.name_en
		RETURNING id;
	`, nameAR, nameEN, slug).Scan(&newID)
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	return &CategorySummary{
		ID:         newID,
		NameAR:     nameAR,
		NameEN:     nameEN,
		Slug:       slug,
		MediaCount: 0,
		FileCount:  0,
	}, nil
}

func (r *Repository) UpdateCategory(ctx context.Context, id int64, nameAR, nameEN, slug string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE categories
		SET
			name_ar = COALESCE(NULLIF($1, ''), name_ar),
			name_en = COALESCE(NULLIF($2, ''), name_en),
			slug = COALESCE(NULLIF($3, ''), slug)
		WHERE id = $4;
	`, strings.TrimSpace(nameAR), strings.TrimSpace(nameEN), strings.ToLower(strings.TrimSpace(slug)), id)
	return err
}

func (r *Repository) DeleteCategory(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	return err
}

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/lib/pq"

	"nexora/server/internal/search"
)

func (r *Repository) ListCollections(ctx context.Context) ([]Collection, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT c.id,c.slug,COALESCE(c.title_ar,''),c.title_en,COALESCE(c.description_ar,''),COALESCE(c.description_en,''),COALESCE(c.artwork_path,''),c.artwork_position,c.accent,COALESCE(c.target_category_slug,''),c.target_filters::text,c.priority,c.is_active,COUNT(ci.media_item_id)::int,COALESCE(array_agg(ci.media_item_id ORDER BY ci.sort_order,ci.media_item_id) FILTER (WHERE ci.media_item_id IS NOT NULL),'{}') FROM collections c LEFT JOIN collection_items ci ON ci.collection_id=c.id GROUP BY c.id ORDER BY c.priority DESC,c.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()
	items := []Collection{}
	for rows.Next() {
		var item Collection
		var filters string
		if err := rows.Scan(&item.ID, &item.Slug, &item.TitleAR, &item.TitleEN, &item.DescriptionAR, &item.DescriptionEN, &item.ArtworkPath, &item.ArtworkPosition, &item.Accent, &item.TargetCategorySlug, &filters, &item.Priority, &item.IsActive, &item.ItemCount, pq.Array(&item.ItemIDs)); err != nil {
			return nil, err
		}
		item.TargetFilters = json.RawMessage(filters)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveCollection(ctx context.Context, id int64, req CollectionRequest) (*Collection, error) {
	if strings.TrimSpace(req.TitleEN) == "" {
		req.TitleEN = req.TitleAR
	}
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if req.Slug == "" || strings.TrimSpace(req.TitleAR) == "" {
		return nil, errors.New("hub slug and Arabic title are required")
	}
	if req.Slug == "" {
		req.Slug = strings.ReplaceAll(strings.ToLower(req.TitleEN), " ", "-")
	}
	if len(req.TargetFilters) == 0 {
		req.TargetFilters = json.RawMessage(`{}`)
	}
	var saved Collection
	query := `INSERT INTO collections (slug,title_ar,title_en,description_ar,description_en,artwork_path,artwork_position,accent,target_category_slug,target_filters,priority,is_active) VALUES ($1,$2,$3,$4,$5,$6,COALESCE(NULLIF($7,''),'center center'),COALESCE(NULLIF($8,''),'violet'),NULLIF($9,''),$10::jsonb,$11,$12) ON CONFLICT (slug) DO UPDATE SET title_ar=EXCLUDED.title_ar,title_en=EXCLUDED.title_en,description_ar=EXCLUDED.description_ar,description_en=EXCLUDED.description_en,artwork_path=EXCLUDED.artwork_path,artwork_position=EXCLUDED.artwork_position,accent=EXCLUDED.accent,target_category_slug=EXCLUDED.target_category_slug,target_filters=EXCLUDED.target_filters,priority=EXCLUDED.priority,is_active=EXCLUDED.is_active,updated_at=CURRENT_TIMESTAMP RETURNING id,slug,title_ar,title_en,description_ar,description_en,artwork_path,artwork_position,accent,COALESCE(target_category_slug,''),target_filters::text,priority,is_active`
	if id > 0 {
		query = `UPDATE collections SET slug=$1,title_ar=$2,title_en=$3,description_ar=$4,description_en=$5,artwork_path=$6,artwork_position=COALESCE(NULLIF($7,''),'center center'),accent=COALESCE(NULLIF($8,''),'violet'),target_category_slug=NULLIF($9,''),target_filters=$10::jsonb,priority=$11,is_active=$12,updated_at=CURRENT_TIMESTAMP WHERE id=` + fmt.Sprint(id) + ` RETURNING id,slug,title_ar,title_en,description_ar,description_en,artwork_path,artwork_position,accent,COALESCE(target_category_slug,''),target_filters::text,priority,is_active`
	}
	var filters string
	err := r.db.QueryRowContext(ctx, query, req.Slug, req.TitleAR, req.TitleEN, req.DescriptionAR, req.DescriptionEN, req.ArtworkPath, req.ArtworkPosition, req.Accent, req.TargetCategorySlug, string(req.TargetFilters), req.Priority, req.IsActive).Scan(&saved.ID, &saved.Slug, &saved.TitleAR, &saved.TitleEN, &saved.DescriptionAR, &saved.DescriptionEN, &saved.ArtworkPath, &saved.ArtworkPosition, &saved.Accent, &saved.TargetCategorySlug, &filters, &saved.Priority, &saved.IsActive)
	if err != nil {
		return nil, err
	}
	saved.TargetFilters = json.RawMessage(filters)
	if _, err := r.db.ExecContext(ctx, `DELETE FROM collection_items WHERE collection_id=$1`, saved.ID); err != nil {
		return nil, fmt.Errorf("clear collection items: %w", err)
	}
	for index, mediaID := range req.ItemIDs {
		if mediaID <= 0 {
			continue
		}
		if _, err := r.db.ExecContext(ctx, `INSERT INTO collection_items (collection_id,media_item_id,sort_order) VALUES ($1,$2,$3) ON CONFLICT (collection_id,media_item_id) DO UPDATE SET sort_order=EXCLUDED.sort_order`, saved.ID, mediaID, index); err != nil {
			return nil, fmt.Errorf("save collection item: %w", err)
		}
	}
	saved.ItemIDs = req.ItemIDs
	return &saved, nil
}

func (r *Repository) DeleteCollection(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collections WHERE id=$1`, id)
	return err
}

func (r *Repository) ListSmartHubsAdmin(ctx context.Context) ([]SmartHub, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT slug,source,scope,title_ar,COALESCE(title_en,''),COALESCE(description_ar,''),COALESCE(description_en,''),COALESCE(artwork_path,''),artwork_position,accent,icon,rule::text,priority,is_active,min_item_count FROM hub_definitions ORDER BY priority DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SmartHub{}
	for rows.Next() {
		var h SmartHub
		var rule string
		if err := rows.Scan(&h.Slug, &h.Source, &h.Scope, &h.TitleAR, &h.TitleEN, &h.DescriptionAR, &h.DescriptionEN, &h.ArtworkPath, &h.ArtworkPosition, &h.Accent, &h.Icon, &rule, &h.Priority, &h.IsActive, &h.MinItemCount); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(rule), &h.Rule); err != nil {
			return nil, err
		}
		h.ID = h.Slug
		items = append(items, h)
	}
	return items, rows.Err()
}

func (r *Repository) SaveSmartHub(ctx context.Context, slug string, req SmartHubRequest) (*SmartHub, error) {
	if strings.TrimSpace(slug) != "" {
		req.Slug = slug
	}
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if req.MinItemCount < 1 {
		req.MinItemCount = 1
	}
	rule, err := json.Marshal(req.Rule)
	if err != nil {
		return nil, err
	}
	var h SmartHub
	var raw string
	err = r.db.QueryRowContext(ctx, `INSERT INTO hub_definitions (slug,source,scope,title_ar,title_en,description_ar,description_en,artwork_path,artwork_position,accent,icon,rule,priority,is_active,min_item_count) VALUES ($1,'editorial',COALESCE(NULLIF($2,''),'all'),$3,$4,$5,$6,$7,COALESCE(NULLIF($8,''),'center center'),COALESCE(NULLIF($9,''),'violet'),COALESCE(NULLIF($10,''),'spark'),$11::jsonb,$12,$13,$14) ON CONFLICT (slug) DO UPDATE SET scope=EXCLUDED.scope,title_ar=EXCLUDED.title_ar,title_en=EXCLUDED.title_en,description_ar=EXCLUDED.description_ar,description_en=EXCLUDED.description_en,artwork_path=EXCLUDED.artwork_path,artwork_position=EXCLUDED.artwork_position,accent=EXCLUDED.accent,icon=EXCLUDED.icon,rule=EXCLUDED.rule,priority=EXCLUDED.priority,is_active=EXCLUDED.is_active,min_item_count=EXCLUDED.min_item_count,updated_at=CURRENT_TIMESTAMP RETURNING slug,source,scope,title_ar,COALESCE(title_en,''),COALESCE(description_ar,''),COALESCE(description_en,''),COALESCE(artwork_path,''),artwork_position,accent,icon,rule::text,priority,is_active,min_item_count`, req.Slug, req.Scope, req.TitleAR, req.TitleEN, req.DescriptionAR, req.DescriptionEN, req.ArtworkPath, req.ArtworkPosition, req.Accent, req.Icon, string(rule), req.Priority, req.IsActive, req.MinItemCount).Scan(&h.Slug, &h.Source, &h.Scope, &h.TitleAR, &h.TitleEN, &h.DescriptionAR, &h.DescriptionEN, &h.ArtworkPath, &h.ArtworkPosition, &h.Accent, &h.Icon, &raw, &h.Priority, &h.IsActive, &h.MinItemCount)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(raw), &h.Rule)
	h.ID = h.Slug
	return &h, nil
}

func (r *Repository) ListSmartHubs(ctx context.Context, scope string) ([]SmartHub, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT slug,source,scope,title_ar,COALESCE(title_en,''),COALESCE(description_ar,''),COALESCE(description_en,''),COALESCE(artwork_path,''),artwork_position,accent,icon,rule::text,priority,is_active,min_item_count FROM hub_definitions WHERE is_active=true AND ($1='' OR scope='all' OR scope=$1) ORDER BY priority DESC,id DESC`, strings.TrimSpace(scope))
	if err != nil {
		return nil, fmt.Errorf("list smart hubs: %w", err)
	}
	defer rows.Close()
	hubs := []SmartHub{}
	seenRules := make(map[string]struct{})
	for rows.Next() {
		var h SmartHub
		var ruleText string
		var minItemCount int
		if err := rows.Scan(&h.Slug, &h.Source, &h.Scope, &h.TitleAR, &h.TitleEN, &h.DescriptionAR, &h.DescriptionEN, &h.ArtworkPath, &h.ArtworkPosition, &h.Accent, &h.Icon, &ruleText, &h.Priority, &h.IsActive, &minItemCount); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(ruleText), &h.Rule); err != nil {
			return nil, fmt.Errorf("decode hub rule %s: %w", h.Slug, err)
		}
		h.ID = h.Slug
		ruleKey := smartHubRuleKey(h.Scope, h.Rule)
		if _, seen := seenRules[ruleKey]; seen {
			continue
		}
		h.MinItemCount = minItemCount
		preview, err := r.ListMediaItems(ctx, listOptionsFromHubRule(h.Rule, ListMediaOptions{Limit: 3, Sort: "rating"}))
		if err != nil {
			return nil, err
		}
		h.ItemCount = preview.Total
		h.PreviewArtwork = hubPreviewArtwork(preview.Items)
		if h.ItemCount >= minItemCount {
			seenRules[ruleKey] = struct{}{}
			hubs = append(hubs, h)
		}
	}
	return hubs, rows.Err()
}

func smartHubRuleKey(scope string, rule HubRule) string {
	canonical := func(values []string) string {
		seen := make(map[string]struct{}, len(values))
		cleaned := make([]string, 0, len(values))
		for _, value := range values {
			value = strings.ToLower(strings.TrimSpace(value))
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			cleaned = append(cleaned, value)
		}
		sort.Strings(cleaned)
		return strings.Join(cleaned, ",")
	}

	return fmt.Sprintf("%s|%s|%s|%s|%d|%d|%.4f", strings.ToLower(strings.TrimSpace(scope)), canonical(rule.Types), canonical(rule.Categories), canonical(rule.TagsAny), rule.YearFrom, rule.YearTo, rule.RatingGTE)
}

func (r *Repository) GetSmartHub(ctx context.Context, slug string) (*SmartHub, error) {
	var h SmartHub
	var ruleText string
	var minItemCount int
	err := r.db.QueryRowContext(ctx, `SELECT slug,source,scope,title_ar,COALESCE(title_en,''),COALESCE(description_ar,''),COALESCE(description_en,''),COALESCE(artwork_path,''),artwork_position,accent,icon,rule::text,priority,min_item_count FROM hub_definitions WHERE slug=$1 AND is_active=true`, strings.TrimSpace(slug)).Scan(&h.Slug, &h.Source, &h.Scope, &h.TitleAR, &h.TitleEN, &h.DescriptionAR, &h.DescriptionEN, &h.ArtworkPath, &h.ArtworkPosition, &h.Accent, &h.Icon, &ruleText, &h.Priority, &minItemCount)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(ruleText), &h.Rule); err != nil {
		return nil, err
	}
	h.ID = h.Slug
	count, err := r.ListMediaItems(ctx, listOptionsFromHubRule(h.Rule, ListMediaOptions{Limit: 3, Sort: "rating"}))
	if err != nil {
		return nil, err
	}
	h.ItemCount = count.Total
	h.PreviewArtwork = hubPreviewArtwork(count.Items)
	if h.ItemCount < minItemCount {
		return nil, sql.ErrNoRows
	}
	return &h, nil
}

func (r *Repository) ListSmartHubMedia(ctx context.Context, slug string, opts ListMediaOptions) (*MediaListResult, *SmartHub, error) {
	hub, err := r.GetSmartHub(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	result, err := r.ListMediaItems(ctx, listOptionsFromHubRule(hub.Rule, opts))
	if err != nil {
		return nil, nil, err
	}
	return result, hub, nil
}

func listOptionsFromHubRule(rule HubRule, opts ListMediaOptions) ListMediaOptions {
	opts.Types = rule.Types
	opts.Categories = rule.Categories
	opts.TagsAny = rule.TagsAny
	opts.YearFrom = rule.YearFrom
	opts.YearTo = rule.YearTo
	opts.RatingGTE = rule.RatingGTE
	return opts
}

func hubPreviewArtwork(items []search.MediaDocument) []string {
	paths := make([]string, 0, len(items))
	for _, item := range items {
		path := item.BannerPath
		if path == "" {
			path = item.PosterPath
		}
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func countHubMatches(items []search.MediaDocument, rule HubRule) int {
	count := 0
	for _, item := range items {
		if matchesHubRule(item, rule) {
			count++
		}
	}
	return count
}

func matchesHubRule(item search.MediaDocument, rule HubRule) bool {
	in := func(values []string, value string) bool {
		for _, v := range values {
			if strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(value)) {
				return true
			}
		}
		return false
	}
	if len(rule.Types) > 0 && !in(rule.Types, item.Type) {
		return false
	}
	if len(rule.Categories) > 0 && !in(rule.Categories, item.CategorySlug) {
		return false
	}
	if rule.YearFrom > 0 && item.ReleaseYear < rule.YearFrom {
		return false
	}
	if rule.YearTo > 0 && item.ReleaseYear > rule.YearTo {
		return false
	}
	if rule.RatingGTE > 0 && item.Rating < rule.RatingGTE {
		return false
	}
	if len(rule.TagsAny) > 0 {
		found := false
		for _, tag := range item.Genres {
			for _, wanted := range rule.TagsAny {
				if strings.EqualFold(strings.TrimSpace(tag), strings.TrimSpace(wanted)) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

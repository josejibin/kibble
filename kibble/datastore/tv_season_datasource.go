package datastore

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/CloudyKit/jet"
	"github.com/indiereign/shift72-kibble/kibble/models"
)

// TVSeasonDataSource - single tv season datasource
// Supports slugs in the /tv/:tvID/season/:seasonID and /tv/:title_slug
type TVSeasonDataSource struct{}

// GetName - name of the datasource
func (ds *TVSeasonDataSource) GetName() string {
	return "TVSeason"
}

// GetEntityType - Get the entity type
func (ds *TVSeasonDataSource) GetEntityType() reflect.Type {
	return reflect.TypeOf(&models.TVSeason{})
}

// Iterator - loop over each film
func (ds *TVSeasonDataSource) Iterator(ctx models.RenderContext, renderer models.Renderer) (errCount int) {

	data := make(jet.VarMap)

	for _, f := range ctx.Site.TVSeasons {
		filePath := ds.GetRouteForEntity(ctx, &f)

		data.Set("tvseason", transformTVSeason(f))
		data.Set("site", ctx.Site)
		errCount += renderer.Render(ctx.Route, filePath, data)
	}

	return
}

// GetRouteForEntity - get the route
func (ds *TVSeasonDataSource) GetRouteForEntity(ctx models.RenderContext, entity interface{}) string {
	o, ok := entity.(*models.TVSeason)
	if ok {
		return ds.GetRouteForSlug(ctx, o.Slug)
	}
	return models.ErrDataSource
}

// GetRouteForSlug - get the route
func (ds *TVSeasonDataSource) GetRouteForSlug(ctx models.RenderContext, slug string) string {

	// supports having tv/:slug/season/:seasonNumber, or any params: :showID, seasonNumber, or :slug
	tvSeason, found := ctx.Site.TVSeasons.FindTVSeasonBySlug(slug)
	if !found {
		return fmt.Sprintf("ERR(%s)", slug)
	}
	s := strings.Replace(ctx.Route.URLPath, ":slug", tvSeason.ShowInfo.TitleSlug, 1)
	s = strings.Replace(s, ":seasonNumber", strconv.Itoa(tvSeason.SeasonNumber), 1)
	s = strings.Replace(s, ":showID", strconv.Itoa(tvSeason.ShowInfo.ID), 1)

	return ctx.RoutePrefix + s
}

// IsSlugMatch - checks if the slug is a match
func (ds *TVSeasonDataSource) IsSlugMatch(slug string) bool {
	return strings.HasPrefix(slug, "/tv/") && strings.Contains(slug, "/season/")
}

package models

import (
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/pressly/chi"
)

// NewRouteRegistry - create a new route registry
func NewRouteRegistry() *RouteRegistry {
	return &RouteRegistry{
		routes: make([]*Route, 0),
	}
}

// FindByName - find the route by the name
func (r *RouteRegistry) FindByName(name string) *Route {
	for _, r := range r.routes {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// FindByTypeAndRouteName - find a route given a type of entity and the route name(optional)
func (r *RouteRegistry) FindByTypeAndRouteName(entityType reflect.Type, routeName string) *Route {
	for _, r := range r.routes {
		if r.ResolvedEntityType == entityType &&
			(routeName == "" || r.Name == routeName) {
			return r
		}
	}
	return nil
}

// FindBySlugAndRouteName - find a route given a slug and the route name(optional)
func (r *RouteRegistry) FindBySlugAndRouteName(slug string, routeName string) *Route {
	for _, r := range r.routes {
		if r.ResolvedDataSouce.IsSlugMatch(slug) &&
			(routeName == "" || r.Name == routeName) {
			return r
		}
	}
	return nil
}

// GetAll - return all routes
func (r *RouteRegistry) GetAll() []*Route {
	return r.routes
}

// GetRouteForEntity - finds the route by the name and type and creates a route from it
func (r *RouteRegistry) GetRouteForEntity(ctx RenderContext, entity interface{}, routeName string) string {

	ctx.Route = r.FindByTypeAndRouteName(reflect.TypeOf(entity), routeName)

	if ctx.Route != nil {
		// fmt.Printf("Found route, name:%s, path: %s\n", foundRoute.Name, foundRoute.URLPath)
		return ctx.Route.ResolvedDataSouce.GetRouteForEntity(ctx, entity)
	}

	return fmt.Sprintf("!Error. Route not found for entity:%s and route name %v", reflect.TypeOf(entity).Name(), routeName)
}

// GetRouteForSlug - finds the route by the name and type and creates a route from it
func (r *RouteRegistry) GetRouteForSlug(ctx RenderContext, slug string, routeName string) string {

	ctx.Route = r.FindBySlugAndRouteName(slug, routeName)

	if ctx.Route != nil {
		// fmt.Printf("Found route, name:%s, path: %s\n", foundRoute.Name, foundRoute.URLPath)
		return ctx.Route.ResolvedDataSouce.GetRouteForSlug(ctx, slug)
	}

	return fmt.Sprintf("!Error. Route not found for slug:%s and route name %v", slug, routeName)
}

// NewRouteRegistryFromConfig - create a new route registry from the config
func NewRouteRegistryFromConfig(config *Config) *RouteRegistry {
	routeRegistry := NewRouteRegistry()

	routeRegistry.routes = make([]*Route, len(config.Routes))

	for i := 0; i < len(config.Routes); i++ {
		route := config.Routes[i]

		route.ResolvedDataSouce = FindDataSource(route.DataSource)
		if route.ResolvedDataSouce != nil {
			route.ResolvedEntityType = route.ResolvedDataSouce.GetEntityType()
		} else {
			fmt.Printf("Unable to find the datasource %s\n", route.DataSource)
		}
		routeRegistry.routes[i] = &route
	}

	return routeRegistry
}

// AddToRouter - adds the route to the router
func (route *Route) AddToRouter(r chi.Router, handler func(w http.ResponseWriter, req *http.Request)) {
	if route.ResolvedDataSouce != nil {
		route.ResolvedDataSouce.RegisterRoutes(r, route, handler)
	} else {
		log.Printf("Route skipped, unknown data source %s\n", route.DataSource)
	}
}

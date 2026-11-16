package sql

import "context"

// WithContext returns a copy of the entity whose queries run under ctx: pass
// the request's context (types.RequestContext, or a context.Context DTO field)
// so a query stops when the request is cancelled or its deadline passes.
func (r *Entity[T]) WithContext(ctx context.Context) *Entity[T] {
	return r.With(r.query.WithContext(ctx))
}

// WithContext returns a copy of the Query whose statements run under ctx.
func (r *Query) WithContext(ctx context.Context) *Query {
	if ctx == nil {
		return r
	}
	clone := *r
	clone.db = &Db{DB: r.db.DB.WithContext(ctx), log: r.db.log}
	return &clone
}

package bot

import (
	"context"

	"github.com/Semior001/newsfeed/app/store"
)

type userKey struct{}

func userFromContext(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(userKey{}).(store.User)
	return u, ok
}

func contextWithUser(ctx context.Context, u store.User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

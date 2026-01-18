package archive

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	celext "buf.build/go/protovalidate/cel"
	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
	"github.com/stolasapp/erato/internal/pagination"
)

const (
	resultsVar = "results"
	thisVar    = "this"
	exprFormat = resultsVar + ".filter(" + thisVar + ", %s)"
)

var (
	categoriesFieldDesc = (&eratov1.ListCategoriesResponse{}).ProtoReflect().Descriptor().Fields().ByName("results")
	entriesFieldDesc    = (&eratov1.ListEntriesResponse{}).ProtoReflect().Descriptor().Fields().ByName("results")
	chaptersFieldDesc   = (&eratov1.ListChaptersResponse{}).ProtoReflect().Descriptor().Fields().ByName("results")
	usersFieldDesc      = (&eratov1.ListUsersResponse{}).ProtoReflect().Descriptor().Fields().ByName("results")

	categoriesCELType = celext.ProtoFieldToType(categoriesFieldDesc, false, false)
	entriesCELType    = celext.ProtoFieldToType(entriesFieldDesc, false, false)
	chaptersCELType   = celext.ProtoFieldToType(chaptersFieldDesc, false, false)
	usersCELType      = celext.ProtoFieldToType(usersFieldDesc, false, false)
)

// Paginator is a [eratov1connect.ArchiveServiceHandler] decorator that applies
// pagination and filtering.
type Paginator struct {
	eratov1connect.ArchiveServiceHandler

	categoriesEnv *cel.Env
	entriesEnv    *cel.Env
	chaptersEnv   *cel.Env
	usersEnv      *cel.Env
}

// NewPaginator decorates inner, applying pagination and filtering to list
// results. This should be called after hydration of the messages.
func NewPaginator(inner eratov1connect.ArchiveServiceHandler) (*Paginator, error) {
	base, err := cel.NewEnv(cel.Lib(celext.NewLibrary()))
	if err != nil {
		return nil, fmt.Errorf("failed to create paginator base CEL environment: %w", err)
	}
	paginator := &Paginator{
		ArchiveServiceHandler: inner,
	}
	if paginator.categoriesEnv, err = initCELEnv(base, categoriesFieldDesc, categoriesCELType, "categories"); err != nil {
		return nil, err
	}
	if paginator.entriesEnv, err = initCELEnv(base, entriesFieldDesc, entriesCELType, "entries"); err != nil {
		return nil, err
	}
	if paginator.chaptersEnv, err = initCELEnv(base, chaptersFieldDesc, chaptersCELType, "chapters"); err != nil {
		return nil, err
	}
	if paginator.usersEnv, err = initCELEnv(base, usersFieldDesc, usersCELType, "users"); err != nil {
		return nil, err
	}
	return paginator, nil
}

// ListCategories satisfies [eratov1connect.ArchiveServiceHandler].
func (p *Paginator) ListCategories(
	ctx context.Context,
	req *connect.Request[eratov1.ListCategoriesRequest],
) (*connect.Response[eratov1.ListCategoriesResponse], error) {
	res, err := p.ArchiveServiceHandler.ListCategories(ctx, req)
	if err != nil {
		return nil, err
	}

	return res, applyPagination(
		ctx,
		req.Msg,
		res.Msg,
		p.categoriesEnv,
		categoriesCELType,
		func(tkn *eratov1.ListCategoriesPaginationToken) {
			results := res.Msg.GetResults()
			if idx := slices.IndexFunc(results, func(cat *eratov1.Category) bool {
				return cat.GetPath() == tkn.GetAfterCategory()
			}); idx != -1 {
				res.Msg.SetResults(results[idx+1:])
				return
			}
			// after_category is not in results, find the next inclusive value
			if idx := slices.IndexFunc(results, func(cat *eratov1.Category) bool {
				return cmp.Less(tkn.GetAfterCategory(), cat.GetPath())
			}); idx != -1 {
				res.Msg.SetResults(res.Msg.GetResults()[idx:])
				return
			}
			// all categories come before after_category alphabetically, return nothing
			res.Msg.SetResults(nil)
		},
		func(size int, token *eratov1.ListCategoriesPaginationToken) *eratov1.ListCategoriesPaginationToken {
			results := res.Msg.GetResults()[:size]
			res.Msg.SetResults(results)
			if token == nil {
				token = &eratov1.ListCategoriesPaginationToken{}
			}
			token.SetAfterCategory(results[size-1].GetPath())
			return token
		},
	)
}

// ListEntries satisfies [eratov1connect.ArchiveServiceHandler].
func (p *Paginator) ListEntries(
	ctx context.Context,
	req *connect.Request[eratov1.ListEntriesRequest],
) (*connect.Response[eratov1.ListEntriesResponse], error) {
	res, err := p.ArchiveServiceHandler.ListEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	// Track remaining entries after cursor slicing to determine if we need to advance to next upstream page
	var entriesAfterCursor int
	return res, applyPagination(
		ctx,
		req.Msg,
		res.Msg,
		p.entriesEnv,
		entriesCELType,
		func(tkn *eratov1.ListEntriesPaginationToken) {
			results := res.Msg.GetResults()
			if idx := slices.IndexFunc(results, func(entry *eratov1.Entry) bool {
				return entry.GetPath() == tkn.GetAfterEntry() &&
					proto.Equal(entry.GetUpdateTime(), tkn.GetStartUpdateTime())
			}); idx != -1 {
				res.Msg.SetResults(results[idx+1:])
				entriesAfterCursor = len(results) - idx - 1
				return
			}
			// after_entry has updated or been removed from the page, start on or after start_update_time
			tknStartTime := tkn.GetStartUpdateTime().AsTime()
			if idx := slices.IndexFunc(results, func(entry *eratov1.Entry) bool {
				entryUpdateTime := entry.GetUpdateTime().AsTime()
				return tknStartTime.Equal(entryUpdateTime) || tknStartTime.Before(entryUpdateTime)
			}); idx != -1 {
				res.Msg.SetResults(res.Msg.GetResults()[idx:])
				entriesAfterCursor = len(results) - idx
				return
			}
			// all entries come before start_update_time - upstream page exhausted, need next page
			res.Msg.SetResults(nil)
			entriesAfterCursor = 0
		},
		func(size int, token *eratov1.ListEntriesPaginationToken) *eratov1.ListEntriesPaginationToken {
			results := res.Msg.GetResults()[:size]
			res.Msg.SetResults(results)
			if token == nil {
				token = eratov1.ListEntriesPaginationToken_builder{
					Page: 1,
				}.Build()
			}
			// If we consumed all remaining entries (not windowing), advance to next upstream page
			if entriesAfterCursor > 0 && entriesAfterCursor <= size {
				token.SetPage(token.GetPage() + 1)
			}
			token.SetAfterEntry(results[size-1].GetPath())
			token.SetStartUpdateTime(results[size-1].GetUpdateTime())
			return token
		},
	)
}

// ListChapters satisfies [eratov1connect.ArchiveServiceHandler].
func (p *Paginator) ListChapters(
	ctx context.Context,
	req *connect.Request[eratov1.ListChaptersRequest],
) (*connect.Response[eratov1.ListChaptersResponse], error) {
	res, err := p.ArchiveServiceHandler.ListChapters(ctx, req)
	if err != nil {
		return nil, err
	}

	return res, applyPagination(
		ctx,
		req.Msg,
		res.Msg,
		p.chaptersEnv,
		chaptersCELType,
		func(tkn *eratov1.ListChaptersPaginationToken) {
			results := res.Msg.GetResults()
			if idx := slices.IndexFunc(results, func(chapter *eratov1.Chapter) bool {
				return chapter.GetPath() == tkn.GetAfterChapter()
			}); idx != -1 {
				res.Msg.SetResults(results[idx+1:])
			}
		},
		func(size int, token *eratov1.ListChaptersPaginationToken) *eratov1.ListChaptersPaginationToken {
			results := res.Msg.GetResults()[:size]
			res.Msg.SetResults(results)
			if token == nil {
				token = &eratov1.ListChaptersPaginationToken{}
			}
			token.SetAfterChapter(results[size-1].GetPath())
			return token
		},
	)
}

// ListUsers satisfies [eratov1connect.ArchiveServiceHandler].
func (p *Paginator) ListUsers(
	ctx context.Context,
	req *connect.Request[eratov1.ListUsersRequest],
) (*connect.Response[eratov1.ListUsersResponse], error) {
	res, err := p.ArchiveServiceHandler.ListUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	return res, applyPagination(
		ctx,
		req.Msg,
		res.Msg,
		p.usersEnv,
		usersCELType,
		func(tkn *eratov1.ListUsersPaginationToken) {
			results := res.Msg.GetResults()
			if idx := slices.IndexFunc(results, func(user *eratov1.User) bool {
				return user.GetPath() == tkn.GetAfterUser()
			}); idx != -1 {
				res.Msg.SetResults(results[idx+1:])
				return
			}
		},
		func(size int, token *eratov1.ListUsersPaginationToken) *eratov1.ListUsersPaginationToken {
			results := res.Msg.GetResults()[:size]
			res.Msg.SetResults(results)
			if token == nil {
				token = &eratov1.ListUsersPaginationToken{}
			}
			token.SetAfterUser(results[size-1].GetPath())
			return token
		},
	)
}

type paginatedRequest interface {
	GetPageToken() string
	GetMaxPageSize() int32
}

type filteredRequest interface {
	paginatedRequest
	GetFilter() string
}

type paginatedResponse[E any] interface {
	proto.Message

	GetResults() []*E
	SetResults(value []*E)
	GetNextPageToken() string
	SetNextPageToken(value string)
}

func applyPagination[Elem any, Tkn proto.Message](
	ctx context.Context,
	req paginatedRequest,
	res paginatedResponse[Elem],
	env *cel.Env,
	resultsType *cel.Type,
	applyPageTokenFn func(tkn Tkn),
	applyPageSizeFn func(size int, tkn Tkn) Tkn,
) error {
	if err := applyToken(req.GetPageToken(), applyPageTokenFn); err != nil {
		return err
	}
	if err := applyFilter(ctx, env, req, res, resultsType); err != nil {
		return err
	}
	return applyPageSize(req, res, applyPageSizeFn)
}

func applyToken[Tkn proto.Message](
	pageTkn string,
	applyFn func(tkn Tkn),
) error {
	if pageTkn == "" {
		return nil
	}
	var tkn Tkn
	tkn = tkn.ProtoReflect().New().Interface().(Tkn) //nolint:forcetypeassert // guaranteed to be the right type
	if err := pagination.FromToken(pageTkn, tkn); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	applyFn(tkn)
	return nil
}

func applyPageSize[Tkn proto.Message, Elem any](
	req paginatedRequest,
	res paginatedResponse[Elem],
	applyFn func(size int, tkn Tkn) Tkn,
) error {
	results := res.GetResults()
	size, currentSize := int(req.GetMaxPageSize()), len(results)
	if size <= 0 || currentSize < size {
		return nil
	}

	var tkn Tkn
	if nextTkn := res.GetNextPageToken(); nextTkn != "" {
		tkn = tkn.ProtoReflect().New().Interface().(Tkn) //nolint:forcetypeassert // guaranteed to be the right type
		if err := pagination.FromToken(nextTkn, tkn); err != nil {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	if tkn = applyFn(size, tkn); tkn.ProtoReflect().IsValid() {
		tknStr, err := pagination.ToToken(tkn)
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		res.SetNextPageToken(tknStr)
	} else {
		res.SetNextPageToken("")
	}

	return nil
}

func applyFilter[Elem any](
	ctx context.Context,
	env *cel.Env,
	req paginatedRequest,
	res paginatedResponse[Elem],
	resultsType *cel.Type,
) error {
	freq, ok := req.(filteredRequest)
	if !ok || freq.GetFilter() == "" || len(res.GetResults()) == 0 {
		return nil
	}
	prog, err := compileFilter(env, resultsType, freq.GetFilter())
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	val, _, err := prog.ContextEval(ctx, map[string]any{
		resultsVar: res.GetResults(),
	})
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	list, ok := val.Value().([]ref.Val)
	if !ok {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("expected list, got %T", val.Value()))
	}
	out := make([]*Elem, len(list))
	for i, el := range list {
		elem, ok := el.Value().(*Elem)
		if !ok {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("expected %T, got %T", (*Elem)(nil), el.Value()))
		}
		out[i] = elem
	}

	res.SetResults(out)
	return nil
}

func compileFilter(env *cel.Env, resultsType *cel.Type, filter string) (cel.Program, error) {
	expr := fmt.Sprintf(exprFormat, filter)
	ast, issues := env.Compile(expr)
	if err := issues.Err(); err != nil {
		return nil, fmt.Errorf("failed to compile filter: %w", err)
	}

	outType := ast.OutputType()
	if !outType.IsExactType(resultsType) {
		return nil, fmt.Errorf("filter expression must return %s but got %s", resultsType.String(), outType.String())
	}

	return env.Program(ast)
}

func initCELEnv(
	base *cel.Env,
	resultsField protoreflect.FieldDescriptor,
	resultsType *cel.Type,
	name string,
) (*cel.Env, error) {
	env, err := base.Extend(append(
		celext.RequiredEnvOptions(resultsField),
		cel.Variable(resultsVar, resultsType),
	)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s CEL environment: %w", name, err)
	}
	return env, nil
}

var _ eratov1connect.ArchiveServiceHandler = (*Paginator)(nil)

/*
Copyright 2022 Natun.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pyexp

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/natun-ai/natun/pkg/api"
	"github.com/qri-io/starlib/bsoup"
	"github.com/qri-io/starlib/encoding/base64"
	"github.com/qri-io/starlib/geo"
	"github.com/qri-io/starlib/hash"
	"github.com/qri-io/starlib/html"
	"github.com/qri-io/starlib/re"
	"github.com/sourcegraph/starlight/convert"
	sJson "go.starlark.net/lib/json"
	sMath "go.starlark.net/lib/math"
	sTime "go.starlark.net/lib/time"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"time"
)

// HandlerFuncName is the name of the function that the use need to implement to handle the request.
const HandlerFuncName = "handler"
const localKeyContext = "go.context"

func init() {
	starlark.Universe["json"] = sJson.Module
	starlark.Universe["time"] = sTime.Module
	starlark.Universe["math"] = sMath.Module
	starlark.Universe["struct"] = starlark.NewBuiltin("struct", starlarkstruct.Make)
	resolve.AllowRecursion = true
	resolve.AllowSet = true

	rer, _ := re.LoadModule()
	hashr, _ := hash.LoadModule()
	geor, _ := geo.LoadModule()
	bsoupr, _ := bsoup.LoadModule()
	base64r, _ := base64.LoadModule()
	htmlr, _ := html.LoadModule()
	starlark.Universe["re"] = rer["re"]
	starlark.Universe["hash"] = hashr["hash"]
	starlark.Universe["geo"] = geor["geo"]
	starlark.Universe["bsoup"] = bsoupr["bsoup"]
	starlark.Universe["html"] = htmlr["html"]
	starlark.Universe["base64"] = base64r["base64"]
}

// Runtime is the starlark runtime for the PyExp.
type Runtime interface {
	Exec(context.Context, ExecRequest) (value any, timestamp time.Time, entityID string, err error)
}

type runtime struct {
	program  *starlark.Program
	builtins starlark.StringDict
	engine   api.Engine
}

// New returns a new PyExp runtime
func New(program string, e api.Engine) (Runtime, error) {
	d := &runtime{
		engine:   e,
		builtins: starlark.StringDict{},
	}

	// This dictionary defines the pre-declared environment.
	d.builtins["f"] = starlark.NewBuiltin("f", d.GetFeature)
	d.builtins["get_feature"] = starlark.NewBuiltin("get_feature", d.GetFeature)
	d.builtins["set_feature"] = starlark.NewBuiltin("set_feature", d.SetFeature)
	d.builtins["update_feature"] = starlark.NewBuiltin("update_feature", d.Update)
	d.builtins["append_feature"] = starlark.NewBuiltin("append_feature", d.AppendFeature)
	d.builtins["incr_feature"] = starlark.NewBuiltin("incr_feature", d.Incr)
	d.builtins["payload"] = starlark.None
	d.builtins["headers"] = starlark.None
	d.builtins["feature_fqn"] = starlark.None
	d.builtins["entity_id"] = starlark.None
	d.builtins["timestamp"] = starlark.None

	// Parse, resolve, and compile a Starlark source file.
	f, p, err := starlark.SourceProgram("<pyexp>", program, d.builtins.Has)
	if err != nil {
		return nil, err
	}

	if p.NumLoads() > 0 {
		return nil, errors.New("pyexp cannot load files")
	}

	if !isHandlerImplemented(f) {
		return nil, fmt.Errorf("`%s` func has not declared and is required by the Natun spec", HandlerFuncName)
	}

	d.program = p
	return d, nil
}

type ExecRequest struct {
	Headers   map[string][]string
	Payload   any
	EntityID  string
	Fqn       string
	Timestamp time.Time
	Logger    logr.Logger
}

func (r *runtime) Exec(ctx context.Context, req ExecRequest) (any, time.Time, string, error) {
	// Prepare globals
	predeclared, err := requestToPredeclared(req, r.builtins)
	if err != nil {
		return nil, time.Now(), "", err
	}

	// Create a Thread and redefine the behavior of the built-in 'print' function.
	thread := &starlark.Thread{
		Name:  req.Fqn,
		Print: func(_ *starlark.Thread, msg string) { req.Logger.WithName("program").Info(msg) },
	}
	thread.SetLocal(localKeyContext, ctx)

	// Execute the program
	globals, err := r.program.Init(thread, predeclared)
	globals.Freeze()

	if err != nil {
		logExexErr(err, req.Logger)
		return nil, time.Now(), "", err
	}

	// Call the handler
	v, err := starlark.Call(thread, globals[HandlerFuncName], nil, nil)
	if err != nil {
		logExexErr(err, req.Logger)
		return nil, time.Now(), "", err
	}

	// Convert and validate the returned value
	ret, ts, eid, err := parseHandlerResults(v)
	if err != nil {
		return nil, time.Now(), "", err
	}
	if req.EntityID != "" && eid != "" && eid != req.EntityID {
		err := fmt.Errorf("execution returned entity id %s, but the request was for entity id %s", eid, req.EntityID)
		return nil, ts, req.EntityID, err
	}
	if req.EntityID == "" && eid == "" {
		return nil, time.Now(), "", fmt.Errorf("this program must return an entity_id along with the value")
	}

	return ret, ts, eid, nil
}

func logExexErr(err error, logger logr.Logger) {
	if err == nil {
		return
	}

	evalErr := &starlark.EvalError{}
	if ok := errors.As(err, &evalErr); ok {
		logger.WithValues("backtrace", evalErr.Backtrace()).Error(evalErr, "execution failed")
	} else {
		logger.Error(err, "execution failed")
	}
}

func requestToPredeclared(req ExecRequest, builtins starlark.StringDict) (starlark.StringDict, error) {
	var payload starlark.Value
	var err error
	if req.Payload == nil {
		payload = starlark.None
	} else {
		if v, ok := req.Payload.(map[string]interface{}); ok {
			req.Payload, err = convert.MakeStringDict(v)
			if err != nil {
				return nil, err
			}
		}
		payload, err = convert.ToValue(req.Payload)
		if err != nil {
			return nil, err
		}
	}

	// Create the globals dict
	globals := starlark.StringDict{}
	// Set builtins types
	for k, v := range builtins {
		globals[k] = v
	}

	// Set per invocation environment for the script
	globals["headers"] = headersToStarDict(req.Headers)
	globals["payload"] = payload
	globals["entity_id"] = starlark.String(req.EntityID)
	globals["feature_fqn"] = starlark.String(req.Fqn)
	globals["timestamp"] = sTime.Time(req.Timestamp)
	return globals, nil
}

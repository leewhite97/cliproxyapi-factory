package protocol

import "factory.local/cliproxyapi-factory/internal/model"

type Route = model.Route

func Resolve(id string) (Route, error) { return model.Resolve(id) }

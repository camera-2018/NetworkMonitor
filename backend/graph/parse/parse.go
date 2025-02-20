package parse

import "github.com/BaiMeow/NetworkMonitor/graph/entity"

type Parser[T entity.DrawType] interface {
	Parse(input any) (T, error)
	CleanUp() error
}

type ParserSpawner[T entity.DrawType] = func(map[string]any) (Parser[T], error)

var registry = make(map[string]any)

func Register[T entity.DrawType](name string, spawnFunc ParserSpawner[T]) {
	registry[name] = spawnFunc
}

func GetSpawner[T entity.DrawType](name string) ParserSpawner[T] {
	return registry[name].(ParserSpawner[T])
}

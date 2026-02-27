package types

type EnvSource interface {
	Load(env Env)
}

type Env interface {
	FromSources(sources ...EnvSource)
	Get(key string) string
	Set(key, value string)

	GetWithDefault(key, defaultValue string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetFloat(key string) float64
}

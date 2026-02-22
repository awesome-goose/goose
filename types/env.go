package types

type EnvSource interface {
	Load(env Env)
}

type Env interface {
	FromSources(sources ...EnvSource)
	Get(key, defaultValue string) string
	Set(key, value string)
}

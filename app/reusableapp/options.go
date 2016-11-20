package reusableapp

// Options specify the available options when creating an App.
type Options struct {
	// The reusable app name. It must not be empty.
	Name string
	// Relative path (to the source file calling New()) to the assets directory.
	// If empty, defaults to "assets". See also the AssetsData field.
	AssetsDir string
	// The baked data string representing the assets.
	//
	// Since reusable apps need to have their non-go files baked, this field
	// is provided as a conveniency in order to simplify reusable app initialization.
	// This field will take precedence over AssetsDir when the following conditions
	// are met:
	//
	// - AssetsData is non-empty
	// - the environment variable GONDOLA_APP_NO_BAKED_DATA doesn't contain the app name (as passed to New in Options)
	//
	// Otherwise, AssetsDir will be used.
	// This is useful while writing reusable apps, so you can start the development server with:
	//
	//	GONDOLA_APP_NO_VFS=MyApp
	//
	// And then your app will use the assets from the filesystem. Once you're ready to
	// commit the changes, you can regenerate the baked assets via gondola bake
	// (usually called from go generate) and users of your app will be able to just
	// go get it.
	AssetsData string
	// Relative path (to the source file calling New()) to the templates directory.
	// If empty, defaults to "tmpl". See TemplatesData and AssetsData.
	TemplatesDir string
	// Same as AssetsData, but for templates.
	TemplatesData string

	// Arbirtrary data than can be retrieved either with Data or AppData
	Data interface{}
	// This field can be used to provide a custom key to store the Data. Note that
	// if this field is non-nil, you must always use AppDataWithKey to retrieve your
	// additional data (Data, AppData, etc... will always return nil).
	DataKey interface{}
}

package domain

type Group struct {
	ID          string
	Name        string
	Description *string
	Label       *string
	Logo        *string
	Page        *string
	Color       *string
	Position    int32
	Concept     string
}

type GroupSummary struct {
	ID   string
	Name string
}

type Category struct {
	ID               string
	GroupID          string
	ParentCategoryID *string
	Name             string
}

type CreateGroupInput struct {
	Name        string
	Description *string
	Label       *string
	Logo        *string
	Page        *string
	Color       *string
	Position    *int32
	Concept     string
}

type ReplaceGroupInput struct {
	Name        string
	Description *string
	Label       *string
	Logo        *string
	Page        *string
	Color       *string
	Position    int32
	Concept     string
}

type CreateCategoryInput struct {
	Name             string
	ParentCategoryID *string
}

type ReplaceCategoryInput struct {
	Name             string
	ParentCategoryID *string
}

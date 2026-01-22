package airyra

import "time"

// ClientOption configures a Client.
type ClientOption func(*clientConfig)

// clientConfig holds the configuration for a Client.
type clientConfig struct {
	host    string
	port    int
	project string
	agentID string
	timeout time.Duration
}

// defaultConfig returns the default client configuration.
func defaultConfig() *clientConfig {
	return &clientConfig{
		host:    "localhost",
		port:    7432,
		timeout: 30 * time.Second,
	}
}

// WithHost sets the server host.
func WithHost(host string) ClientOption {
	return func(c *clientConfig) {
		c.host = host
	}
}

// WithPort sets the server port.
func WithPort(port int) ClientOption {
	return func(c *clientConfig) {
		c.port = port
	}
}

// WithProject sets the project name.
func WithProject(project string) ClientOption {
	return func(c *clientConfig) {
		c.project = project
	}
}

// WithAgentID sets the agent ID used for task ownership.
func WithAgentID(agentID string) ClientOption {
	return func(c *clientConfig) {
		c.agentID = agentID
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = timeout
	}
}

// CreateTaskOption configures a CreateTask call.
type CreateTaskOption func(*createTaskOptions)

// createTaskOptions holds options for creating a task.
type createTaskOptions struct {
	description *string
	priority    *int
	parentID    *string
}

// WithDescription sets the task description.
func WithDescription(desc string) CreateTaskOption {
	return func(o *createTaskOptions) {
		o.description = &desc
	}
}

// WithPriority sets the task priority.
func WithPriority(priority int) CreateTaskOption {
	return func(o *createTaskOptions) {
		o.priority = &priority
	}
}

// WithParentID sets the parent task ID.
func WithParentID(parentID string) CreateTaskOption {
	return func(o *createTaskOptions) {
		o.parentID = &parentID
	}
}

// UpdateTaskOption configures an UpdateTask call.
type UpdateTaskOption func(*updateTaskOptions)

// updateTaskOptions holds options for updating a task.
type updateTaskOptions struct {
	title       *string
	description *string
	priority    *int
}

// WithTitle sets the task title for update.
func WithTitle(title string) UpdateTaskOption {
	return func(o *updateTaskOptions) {
		o.title = &title
	}
}

// WithUpdateDescription sets the task description for update.
func WithUpdateDescription(desc string) UpdateTaskOption {
	return func(o *updateTaskOptions) {
		o.description = &desc
	}
}

// WithUpdatePriority sets the task priority for update.
func WithUpdatePriority(priority int) UpdateTaskOption {
	return func(o *updateTaskOptions) {
		o.priority = &priority
	}
}

// ListTasksOption configures a ListTasks call.
type ListTasksOption func(*listTasksOptions)

// listTasksOptions holds options for listing tasks.
type listTasksOptions struct {
	status  string
	page    int
	perPage int
}

// defaultListTasksOptions returns the default list options.
func defaultListTasksOptions() *listTasksOptions {
	return &listTasksOptions{
		page:    1,
		perPage: 20,
	}
}

// WithStatus filters tasks by status.
func WithStatus(status TaskStatus) ListTasksOption {
	return func(o *listTasksOptions) {
		o.status = string(status)
	}
}

// WithPage sets the page number (1-indexed).
func WithPage(page int) ListTasksOption {
	return func(o *listTasksOptions) {
		o.page = page
	}
}

// WithPerPage sets the number of items per page.
func WithPerPage(perPage int) ListTasksOption {
	return func(o *listTasksOptions) {
		o.perPage = perPage
	}
}

# Resource Access

## Goals

- Can access resources *directories* content like fetching from an api.
- Can limit access to resource.

## Description
Resource is any directory/folder which contains content.
it is created in 2 conditions
- Uploading an object to a specific directory/folder using `key` field.
    - When `key` is manually givin in the api `post` request.
    - Create new directory with path and name `Dir(key)`.
    - Create new resource for the same directory to identify it.

- Uploading multiple objects at once to a givin `subpath`.
    - When `subpath` is sent with multiple objects.
    - We create this `subpath` in the givin bucket.
    - Create new resource for the same `subpath` directory to identify it.

### Resource
id `string` *unique* *indexed*
uuid `string` *unique* *indexed*
name `string` *unique* *indexed*
bucket `string` *indexed*
created_at `date`
updated_at `date`


### Code Implementation

#### Models
```go
type Resource struct {
    mgm.DefaultModel
    UUID string
    Name string
    Bucket string
}

type ObjectFile struct {
	Bucket   string
	File     *os.File
	Object   *Object
	Resource *Resource
}
```

#### Methods
```go
func (r *Resource) CreateIndexes() error

func (r *Resource) Path() string
func (r *Resource) S3Path() string

func Find(r *Resource) error
func FindWithS3(s3 *S3Path, r *Resource) error
func FindResource(r *Resource) (*Resource, error)

func (r *Resource) Create() error
func FindOrCreateResource(r *Resource) error
func createResource(r *Resource) error

func (r *Resource) Find() (*Resource, error)
func findResource(r *Resource) (*Resource, error)

func (r *Resource) Delete() error
func deleteResource(r *Resource) error

func (r *Resource) Exists() (bool, error)
func resourceExists(r *Resource) (bool, error)


func ListObjects(s3 *S3Path) ([]*ObjectFile, error)
func getObject(r *Resource, filename string) (*ObjectFile, error)

```

#### Sharing
*TODO: to be created*
```go

```
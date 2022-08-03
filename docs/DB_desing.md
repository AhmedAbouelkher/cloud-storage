# Design

## Models

### Bucket
* id `integer`
* name `string`
* path `string`

### Object
* id `integer`
* uuid `string`
* title `string`
* directory `string`
* type `string`
* size `integer`
* created_at `data`
* updated_at `data`

---

## API

### Misc
```go
GetStorage() (string, error)
CreateFile(p string, access string) (string, error)
CreateDir(dir string) (string, error)
Exists(p string) (bool, error)
TempFile(p string) (*io.Writer, error)
OpenFile(p string) (*io.Writer, error)
RemoveFile(p string) error
RemoveDir(p string) error
CreateDir(p string) error
OpenMemoryFile() (*io.Writer, error)
```

## Bucket
```go
CreateBucket(name string) (string, error)
DeleteBucket(name string) error
RenameBucket(name string, new string) error
SaveObject(f *io.Writer, p string) error
DeleteObject(id string) error
FetchObject(id string) (*io.Writer, error)
```

## Object
```go
ParsePath(p string) (string, error)
GetUUID(p string) (string, error)
Retrieve(uuid string) (*io.Writer, error)
Save(o *Object, p string) (*io.Writer, error)
```
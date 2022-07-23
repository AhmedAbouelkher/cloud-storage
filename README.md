# Storage Service

### Goals

* Create storage buckets.
* Store object directly.
* Store objects in specific directories in bucket.
* Give every object a unique id to fetch with.
* [FUTURE] authorize access to buckets, objects and directories..

#### Storage Buckets
Bucket is like a directory with access control.

* Create new bucket using name
* Save objects to bucket.
* Save objects in specific directory inside the givin bucket (if exists).
* [FUTURE] Limit access for bucket

#### Storage Objects
Objects are unstructured data, ex: videos, images, audio, etc..

* Objects can ba saved directory to bucket.
* Each object has a unique UUID to reference.
* [FUTURE] Objects size can be limited.
* Can be shared

##### Objects Directories
Directory is a folder to save the givin object in and exists in a givin bucket.

* Files can be saved directory to a directory by attaching the directory name when requesting object save.

##### Sharing Session
Will control the objects sharing aspect and authorization.

* Customer can create *ONLY* one session per object with specific TTL time in secondes (*max 24hrs*)


## What is next?

- [X] Add full api validation.
- [X] Rate limit requests.
- [X] Upload size limit to 1 MB.
- [X] Make Sharable link dynamic with current domain.
- [X] Lock some resources to local usage only.
- [X] Mutate bucket name and return the new one.
# Direct Access

The goal is to directly access objects using their cloud url with a path structure like S3.
example:
```url
s3://{bucket}/{resource}/{path/to/object}
```

## Key Notes

- Access can be limited using a policy (to know more [policy](#policy))

- User can access the object directly using
    - Bucket name.
    - Directory/Resource path.
    - Object real title *without any url encoding* (to know more see [Accessing](#accessing))

## Accessing

To access the required object we should first parse it to extract `bucket` name, `resource` and `object` title *with extension*.

### Steps to serve

- Parse the `S3` url.

> Next step should be matching the stored [policy](#policy) to grant access to the request resource object. *Will be implemented soon*

- Look for the givin resource if it exists or not.
    - If exists
        - Look for the givin object if it exists or not.
    - If not
        - Throw an error and exit.
- Looking for an object can be done using its name *without any url encoding*.
- Return the requested object as a file.


### Url Structure

Url will be like the first example `s3://{bucket}/{resource}/{path/to/object}` very like [AWS S3](https://aws.amazon.com/s3/)

#### Conditions

- Must be url encoded.
- Must start with `s3://` and end with the object name *with extension*.
 
## Policy [TO BE DONE]
Represents a very simple contract to handle external requests to the required resource object. 
We are using it instead of generating `ObjectSharingSession` for each object in the requested directory/resource.

### Policy Model
- Tag `string` [*indexed*, *unique*, *uuid v4*]
- TTL `int`
- Expiry Date `date`
- Resource `string` *indexed*
- Methods `[]string`
- Metadata `map[string]any`

```go
func (s *AccessPolicy) CheckExpiration() bool
//TODO: to be continued
```
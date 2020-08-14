# Encoding and evolutiion

## Formats for encoding

- XML and CSV cannot distinguish between a number and a string
- JSON can distinguish numbers and strings, but cannot distinguish integers and floating numbers
- JSON and XML do not support binary strings, sometimes people use base64 encode the binary strings

### Binary encoding

JSON and XML are not compact and fast compared to binary formats. So people think of encode JSON or XML in binary format, but they need to include all object field names within the encoded data since it does not use prescribed schema.

For example:

``` json
{
    "userName" : "Martin" ,
    "favoriteNumber" : 1337 ,
    "interests" : [ "daydreaming" , "hacking" ]
}
```

![json-binary-encoding](./resources/json-binary-encoding.png)

### Thrift and Protocal Buffers

- binary encoding libraries
- no field keys in the encoded data
- has tag, type, length and value for each field

``` thrift
struct Person {
    1 : required string userName,
    2 : optional i64 favoriteNumber,
    3 : optional list < string > interests
}
```

![thrift](./resources/thrift.png)
![thrift-compact](./resources/thrift-compact.png)

``` protobuf
message Person {
    required string user_name = 1 ;
    optional int64 favorite_number = 2 ;
    repeated string interests = 3 ;
}
```

![protobuf](./resources/protobuf.png)

#### New code has a new field added

``` protobuf
message Person {
    required string user_name = 1 ;
    optional int64 favorite_number = 2 ;
    repeated string interests = 3 ;
    optional int64 user_age = 4;
}
```

Old code can read the record that is written in new code by simply ignore the tag 4.

New code can read the record that is written in old code by simply setting the field 4 with its default value. **Important: New field cannot be requried. The tags of old field cannot be changed.**

#### New code has removed a field

``` protobuf
message Person {
    required string user_name = 1 ;
    repeated string interests = 3 ;
}
```

Old code can read the record that is written in new code by simply setting the field 4 with its default value.

New code can read the record that is written in old code by simply ignore the tag 4.

#### Changing datatype

This operation might cause the value to lose prcision or get truncated.

### Avro

Avro is another binary encoding format that is differnt from Protobuf and Thrift. It also uses schema to specify the data structure of data being encoded.

``` Avro
record Person {
    string userName ;
    union { null , long } favoriteNumber = null ;
    array < string > interests ; }
```

- No tag numbers in schema. The encoding simply consists of values concatenated together
- No `optional` or `requried` markers, it has `union` types and default values instead

![avro](./resources/avro.png)

If we do not have the tag number to identify each fields, the binary data can only be decoded in the correct order only if the code reads the data uses exactly the same schema. **However Avro does not require the schema to be the same, only requries they are compatible.**

![avro-schema-revolution](./resources/avro-schema-revolution.png)

- Avoro lib resolves the diffs by comparing the writer's schema and reader's schema side by side, and translate the data from writer's schema to reader's schema
- Reader looks for the field from writer's schema, if it does not exist then set the default value
- Writer's field which does not exsit in reader's schema, it will be ignored
- In order to guarantee the forward and backward compatibility, only fields with default values could be added or removed

### How does reader know the writer's schema



### Merits of Schemas

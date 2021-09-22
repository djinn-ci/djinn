[Prev](/api/cron) - [Next](/api/invites)

# Images

* [The image object](#the-image-object)
* [List images for the authenticated user](#list-images-for-the-authenticated-user)
* [Create an image for the authenticated user](#create-an-image-for-the-authenticated-user)
* [Get an individual image](#get-an-individual-image)
* [Delete an image](#delete-an-image)

## The image object

<div class="api-section">
<div class="api-doc">

**Attributes**

---

**`id`** `int` - Unique identifier for the image.

---

**`author_id`** `int` - ID of the user who created the image.

---

**`user_id`** `int` - ID of the user the image belongs to.

---

**`namespace_id`** `int` `nullable` - ID of the namespace the image belongs to
if any.

---

**`name`** `string` - The name of the image.

---

**`created_at`** `timestamp` - The RFC3339 formatted string at which the image
was created.

---

**`url`** `string` - The API URL for the image object itself.

---

**`author`** `object` `nullable` - The [user](/api/user#the-user-object) who
authored the image, if any.

---

**`user`** `object` `nullable` - The [user](/api/user#the-user-object) of the
image, if any.

---

**`namespace`** `object` `nullable` - The
[namespace](/api/namespaces#the-namespace-object) the image belongs to, if any.

</div>
<div class="api-example">

**Object**

    {
        "id": 1,
        "author_id": 1,
        "user_id": 1,
        "namespace_id": null,
        "name": "go-dev",
        "created_at": "2006-01-02T15:04:05Z",
        "url": "{{index .Vars "apihost"}}/images/1",
        "user": {
            "id": 1,
            "email": "me@example.com",
            "username": "me",
            "created_at": "2006-01-02T15:04:05Z"
        }
    }

</div>
</div>

## List images for the authenticated user

<div class="api-section">
<div class="api-doc">

This will list all of the images that the currently authenticated user has
access to. This requires the explicit `image:read` permission.

**Parameters**

---

**`search`** `string` - Get the images with names like the given value.

**Returns**

Returns a list of [images](/api/images#the-image-object). The list will be
paginated to 25 images per page and will be ordered by the most recently
created image first. If the images were paginated, then the pagination
information will be in the response header `Link` like so,

    Link: <{{index .Vars "apihost"}}/images?page=1>; rel="prev",
          <{{index .Vars "apihost"}}/images?page=3>; rel="next"

</div>
<div class="api-example">

**Request**

    GET /images

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/images


    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/images?search=go-dev

</div>
</div>

## Create an image for the authenticated user

<div class="api-section">
<div class="api-doc">

This will create an image for the currently authenticated user. This requires
the explicit `image:write` permission.

**Parameters**

---

**`name`** `string` - The name of the image to create.

---

**`namespace`** `string` - The namespace to put the image in.

The contents of the image file should be sent in the body of the request. It
must by a QCOW2 formatted image.

**Returns**

Returns the [image](/api/images#the-image-object). It returns an
[error](/api#errors) if any of the parameters are invalid, or if an internal
error occurs.

</div>
<div class="api-example">

**Request**

    POST /images

**Body**

The body of this request should be the contents of the image file being created.

**Examples**

    $ curl -X POST \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d "@alpine.qcow2" \
           {{index .Vars "apihost"}}/images?name=alpine


    $ curl -X POST \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           -d "@alpine.qcow2" \
           {{index .Vars "apihost"}}/images?name=alpine&namespace=djinn

</div>
</div>

## Get an individual image

<div class="api-section">
<div class="api-doc">

This will get the given image, this requires the explicit `image:read`
permission.

If the `Accept` header is set to `application/x-qemu-disk` then the response
body will be the contents of the image file.

**Returns**

Returns the [image](/api/images#the-image-object). If requested with the
`Accept` header set to `application/octet-stream` then this will download the
image itself.

</div>
<div class="api-example">

**Request**

    GET /images/:image

**Examples**

    $ curl -X GET \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/images/3

    $ curl -X GET \
           -H "Accept: application/octet-stream" \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/images/3

</div>
</div>

## Delete an image

<div class="api-section">
<div class="api-doc">

This will delete the given image, this requires the explicit `image:delete`
permission.

**Returns**

This returns no content in the response body.

</div>
<div class="api-example">

**Request**

    DELETE /images/:image

**Examples**

    $ curl -X DELETE \
           -H "Authorization: Bearer 1a2b3c4d5f" \
           {{index .Vars "apihost"}}/images/3

</div>
</div>

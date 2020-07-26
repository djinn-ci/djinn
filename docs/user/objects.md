# Objects

Objects are files that can be placed into a build environment just after the
creation of the build's driver.

* [Creating an object](#creating-an-object)
* [Using an object](#using-an-object)

## Creating an object

Objects a created from the *Objects* link the dashboard's sidebar, and by
clicking the *Create* button in the top right hand corner.

Objects can be any file that is within the size limit configured by the Djinn
server.

Objects can be grouped in a namespace for use by other users who have access to
that same namespace. If the namespace being specified does not exist during
image creation then it will be created on the fly.

## Using an object

Once an object has been uploaded you can use it simply by specifying the name
you gave that object in the build manifest via the `objects` property,

    objects:
    - my-object

you can use the `=>` notation to specify the name of the object during placement
and the location where it should be placed.

    objects:
    - my-object => /var/lib/object

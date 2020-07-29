[Prev](/user/manifest) - [Next](/user/objects)

# Images

An image is what is used by the Docker and QEMU drivers to setup the build
environment for build execution. A user can upload custom images to use for
their builds. Right now Djinn only supports the uploading of QEMU driver
images.

>**Note:** Since the Docker driver uses Docker Hub to download images, you can
specify any image for this driver to use, and as long as the Djinn server can
talk to the Docker Hub, that image will be downloaded and used.

* [Creating an image](#creating-an-image)
* [Using a custom image](#using-a-custom-image)

## Creating an image

Images a created from the *Images* link the dashboard's sidebar, and by clicking
the *Create* button in the top right hand corner.

Images for the QEMU driver are expected to be QCOW2 (v3) image file. It is
expected for the `root` user to have no password and for SSH access to be
configured for the `root` user.

Images can be grouped in a namespace for use by other users who have access to
that same namespace. If the namespace being specified does not exist during
image creation then it will be created on the fly.

## Using a custom image

Once an image has been uploaded you can use it by simply specifying its name
in the build manifest,

    driver:
        type: qemu
        image: my-image

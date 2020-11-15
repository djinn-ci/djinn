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
  * [Preparing a QEMU image](#preparing-a-qemu-image)
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

### Preparing a QEMU image

First create create a new image file using the `qemu-img` command with the
format of `qcow2`. We recommend keeping the size of this image smaller than 20GB.

    $ qemu-img create -f qcow2 my-image.qcow2 10G

With this image created you can now install an operating system of your choice,
and prepare it for use in Djinn as a custom image.

    $ qemu-system-x86_64 -sdl \
        -m 4096 \
        -cdrom ubuntu.iso \
        -net nic,model=virtio \
        -drive file=my-image.qcow2,media=disk,if=virtio

Once the operating has been installed we can go about configuring the image for
use. The QEMU driver in Djinn uses a passwordless `root` user to connect to the
machine to perform jobs, so we need to ensure that the `root` user has no
password.

    $ passwd -d

Next, we need to configure the SSH server to allow for a root login, and to
allow password authentication. Below is the typical SSH configuration used
by the base images provided by Djinn.

    # Authentication:
    PermitRootLogin yes
    
    PasswordAuthentication yes
    PermitEmptyPasswords yes
    PubkeyAuthentication no
    
    # Change to no to disable s/key passwords
    ChallengeResponseAuthentication no
    GSSAPICleanupCredentials no
    UsePAM yes
    
    # Accept locale-related environment variables
    AcceptEnv *
    
    PermitUserEnvironment yes
    
    # override default of no subsystems
    Subsystem       sftp    /usr/libexec/openssh/sftp-server

## Using a custom image

Once an image has been uploaded you can use it by simply specifying its name
in the build manifest,

    driver:
        type: qemu
        image: my-image

# Namespaces

Namespaces allow you to organize builds and their resources. Users can be
invited to namespaces as a collaborator to work with.

* [Children](#children)
* [Visibility](#visibility)
* [Resources](#resources)
* [Collaborators](#collaborators)

## Children

A namespace can have child namespaces within them for further organization
of builds and build resources. Namespaces can only have a maximum depth of 20.

## Visibility

A namespace can have three levels of visibility as detailed below,

| Level    | Description                                                   |
|----------|---------------------------------------------------------------|
| Private  | Only the creator and any collaborators can see the namespace. |
| Internal | Only users logged in can see the namespace.                   |
| Public   | Anyone can see the namespace.                                 |

## Resources

Namespaces are primarily used for grouping together related builds, and any
resources that you want shared across these builds. Detailed below are the
different resources that can be grouped into a namespace,

* [Images](/user/images)
* [Objects](/user/objects)
* [Variables](/user/variables)
* [Keys](/user/keys)

during the creation of each of these resources you will be given the option to
specify a namespace to put it in. If the given namespace doesn't exist during
resource creation then said namespace will be created on the fly, with the
default visibility of Private.

## Collaborators

Collaborators in a namespace will have the ability to add and remove resources
to and from a namespace, as well as the ability to submit builds to a namespace.
Collaborators are added to a namespace via invites. Only the owner the namespace
will be able to send invites to other users. A user must accept their invite
before they are added to a namespace as a collaborator.

All collaborators will have access to the root namespace, and all of its
children. You cannot silo off collaborators into sub-namespaces. Only the owner
of the namespace will be able to remove collaborators from it.

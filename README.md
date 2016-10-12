# Smutje

NOTE: This is work in progress! It's neither clean nor fully working.

A simple provisioning tool. Using bash scripts and a simple caching layer to
only execute what was not yet executed or has changed.

The idea is to use a markdown dialect to describe the entity to provision. This
way documentation and code are one and (hopefully) easy to read. It's assumed
you're familiar with how markdown works.


## A Simple Example First

Okay, an example first. The following would provision a host (named
"www.example.com" with IP 192.168.1.1) with an nginx server.

	# Resource: www.example.com [example]
	This is a small provisioning example, that will install an example web server.
	
	> Address: 192.168.1.1
	
	## Package: Base Configuration [base]
	First we'll update the package repository
	
		apt-get update
		apt-get upgrade -y
	
	## Package: NGinx Installation [nginx-inst]
	Install the nginx package
	
		apt-get install -y nginx
	
	## Package: NGinx Configuration [nginx-conf]
	Send our custom configuration file to the host.
	
		:write_file nginx_conf /etc/nginx/sites-available/www.example.com root 0644
	
	Enable it after disabling the default configuration.
	
		rm /etc/nginx/sites-enabled/default
		ln -s /etc/nginx/sites-available/www.example.com /etc/nginx/sites-enabled/www.example.com

This is just a teaser to what is actually possible. More examples can be found in
the examples subdirectory.


## Glossary

When talking about all this some common nomenclature might help. So here we go:

* **provisioning**: The process of making a host, VM or zone do, what it
  is supposed to do.
* **host**: Some bare metal machine.
* **vm**: A virtual machine running on some hypervisor like xen, kvm, or
  vmware.
* **zone**: An encapsulated OS process. Think of Linux containers (docker), BSD
  jails, Solaris zones, etc.

In the context of smutje the following terms are important:

* **resource**: This is the something to be provisioned. For a VM or zone this
  includes the blueprint that describes how to create the resource on a
  specific hypervisor. This of course isn't required for hosts. The
  specification of what and how to provision the resource is done using
  packages.
* **package**: Each package is an ordered list of scripts, i.e. it is an
  intermediate abstraction layer. Each package has its own caching. If a script
  changed, all the following scripts in this package will be executed again,
  but following packages are not effected.
* **script**: This is something executed during provisioning of a resource. It
  can either be some bash script, or some smutje specific commands (like
  writing a file on the target).
* **template**: A template is larger abstraction. It is a set of packages that
  can be used to reuse a set of packages for different resources.
* **attributes**: Attributes are used to configure values that are reused often
  or should be changeable (like the version of an installed software package).


## The Markdown Dialect

For each dish there is a markdown file. This file has some semantics above the
markdown syntax to be described here.


### Titles

Titles are used to define what the following content describes. It contains
three elements: a type (what is contained in the section), a name and an
identifier. It always has the form `<Type>: <Name> [<Identifier>]` like in the
following examples:

	# Resource: www.example.org [example]
	## Package: Network Configuration [net_cfg]
	## Include: ./service.smd [service]

The identifier is used in logging and the combination of the respective
identifiers is used as key to the caching layer, i.e. changing an entities
identifier will break caching for all packages below this entity.


### Quotes

Quotes are used to define attributes. Each line defines a key value pair
separated by a colon and whitespace.

	> Key1: Value 1
	> Key2 :   Value 2
	>  Key3   :Value3

Those key-value pairs will be available in the code blocks described next,
using Go's [template mechanism](https://golang.org/pkg/text/template/).


### Code Blocks

Code blocks are used to define scripts. Each block must be indented by same
amount of whitespace (the block's first line defines the amount). Additional
whitespace is possible in following lines and is added to the script.

Lines where the first character is a colon are considered smutje scripts,
otherwise it is bash script. Both can't be mixed in one script!

Each block is a line in the caching layer, i.e. the blocks are atomic regarding
the caching.


## Packages And Scripts

This section describes how to actually describe the steps necessary in
provisioning. The basic building block is a package. Each package contains a
list of scripts. If one script changes all succeeding will be executed again.
If nothing changes nothing will be executed. This has two benefits:

* while developing new packages changes can be tested easily (incremental
  development is possible)
* this allows for update of parts of the provisioned entity

But always keep in mind, that caching will produce snowflakes. You can't be
sure what the actual state is. Especially with VMs and zones it would be much
better to recreate the instance.

Each package contains code blocks. Each code block will be executed as script
on the remote host and cached accordingly. If something changed it (and all
following scripts) will be executed again.

The following two subsections describe the code blocks possible.


### Bash Script Code Block

This is just bash script. Nothing special. Just keep in mind we're going non
interactive. This might require some additional thought.

All scripts are rendered prior to execution, so you can use the go template
language to access the dishes attributes:

	echo "The version is set to {{ .Version }}"

would be rendered to

	echo "The version is set to 1.0"

if the attribute "Version" has the value `1.0`.


### Smutje Script Code Block

Some things require special commands, like sending files to the machine to be
provisioned. This could be embedded in the script, but that would neither be
readable and add a plethora of problems for binary files.

This is why there are some special commands available:

* `write_file`: This is used to send the content of a given file to the
  provisioned entity. For example `write_file foo /tmp/bar peter 0600` would
  read the file `foo`, send the content to the file `/tmp/bar`, set ownership
  to `peter` and set the permissions to `0600`. Please note that if an owner is
  specified, the permissions must be given, too. Giving neither will use the
  defaults.
* `write_template`: Uses the `write_file` logic but will send the file's
  content through the template engine first, i.e. you can again use the dish's
  attributes in the content.
* `jenkins_artifact`: Given the information for a jenkins host and job it will
  download the artifact if it changed since the last run using the artifacts
  fingerprint.

The command line itself is rendered with the template engine, i.e. again the
attributes can be used.


## Templates

A template is used to modularize the provisioning steps. Contrary to resources
it can't be provisioned by itself. It must be included in resources. It may
include other templates though!

The following example shows a template definition:

	# Template: An Example Template [tmpl]
	> TmplAttribute: Value
	
	## Package: A Template Pkg [tpkg]
		echo "inside a template pkg {{ .TmplAttribute }}"
		echo "and a free attribute {{ .FreeAttribute }}"

To use the template it must be included in a resource:

	# Resource: Some Example Host [example]
	
	## Include: example_tmpl.smd [tmpl_inc]
	> FreeAttribute: ValueOfFreeAttribute

Please note that the name of the `Include` section must be the (relative) path
of the template. The `FreeAttribute` specified in the `Include` section is
available in the template instance.


## Resources

Resources are the entity that is actually provisioned. It is either a host, a
VM or a zone. For a VM or zone a `Blueprint` section is required that contains
a description of the resource to create, that is understood by the respective
hypervisor. The hypervisor is specified using the `Hypervisor`, attribute of
the resource. If it is not set, the resource is considered to be existing and
reachable using SSH, which in turn is configured by the `Address` and `Username`
attributes.

If the `Address` attribute is not given, the resource's name and id are checked
whether they can be resolved to an IP using DNS. If no `Username` was specified
`root` will be used.


## Tools

There are two binaries included in smutje:

* **smutje** itself is the binary to provision a given resource. The parameter
  is the file containing the resource.
* **smd-fmt** is a formatter for smutje resource and template definition files.
  It will print out a canonical form of the script and might be a good first
  indicator for problems in these files (like wrong whitespace).



## Requirements

If `sudo` is required (aka instance is connected to using a non root user) then
the asking for password should be disabled:

    echo "<username> ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/90-nopassword
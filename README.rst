noble child
================

On `Intel edison <http://www.intel.com/content/www/us/en/do-it-yourself/edison.html>`_, we can not use `paypal/gatt <http://github.com/paypal/gatt>`_ because the kernel is old(3.10). We can patche the kernel but it is difficult for some people.

This library aims to communicate `noble <https://github.com/sandeepmistry/noble>`_ child process which uses the Bluez library.
And also this has a github.com/paypal/gatt compatible API, after you can use `HCI_CHANNEL_USER` mode, just replace import.

Pre-requirement
----------------

You shoud install noble before use this library.

::

  cd /path/somewhere/
  npm install noble

After install noble,   

Set ``NOBLE_TOPDIR`` environment variable to top of the ``node_modules``. 

For example, when you hit ``npm install noble`` at ``/path/somewhere/``, these two files will be created at

::

   /path/somewhere/node_modules/noble/build/Release/l2cap-ble
   /path/somewhere/node_modules/noble/build/Release/hci-ble

In this case, you should set ``NOBLE_TOPDIR`` to ``/path/somewhere/``.

Tips
+++++++

noblechild also searchs your ``$HOME`` and the path under the executable binary.


How to use
--------------

see example/main.go.

Almost same as paypal/gatt but, ``NewDevice`` and ``d.Hanle`` should be use ``noblechild`` functions.


License
----------

APL 2.0

Copyright (C) 2015 WAKAYAMA Shirou shirou.faw@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

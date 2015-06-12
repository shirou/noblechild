noble child process interface
==============================

- hci-ble
- l2cap-ble

hci-ble
------------------------

Scanを行う

https://github.com/sandeepmistry/noble/blob/master/src/hci-ble.c


- `NOBLE_HCI_DEVICE_ID` 環境変数で hciDeviceId を上書きできる


API
+++++

SIGUSR1
  start scan, filter
SIGUSR2
  start scan, no filter
SIGHUP
  stop scan

もしIDが見つかったら、以下の形式のテキストを標準出力にprintする

event %s, (publicかrandom), 
rssi  

イベント
+++++++++

- stateChange
- scanStart
- scanStop
- discover

  
l2cap-ble
------------------------

connectなどを行う

https://github.com/sandeepmistry/noble/blob/master/src/l2cap-ble.c

最初に以下の文が表示される

printf("info using %s@hci%i\n", controller_address, hciDeviceId);
printf("connect success\n");


addressとaddressTypeを指定して子プロセスを呼び出す

::

  spawn("l2cap-ble", address, addressType)


API
++++++

標準出力
  connect success
  disconnect
  rssi = (.*)
  security = (.*)
  write = (.*)
  data (.*)  
    "%02x"となる
標準入力
  データ書き込み

シグナル

SIGHUP
  disconnectする
SIGUSR1
  updateRssi
  現在のRSSIが標準出力から得られる
SIGUSR2
  upgradeSecurity
  現在のsecurityが標準出力から得られる

  

イベント
++++++++

- connect
- disconnect
- mtu
- rssi
- servicesDiscover
- includedServicesDiscover
- characteristicsDiscover
- read
- write
- broadcast
- notify
- notification
- descriptorsDiscover
- valueRead
- valueWrite
- handleRead
- handleWrite
- handleNotify



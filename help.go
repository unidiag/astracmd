package main

const HelpText = `[::b]astracmd help[::-]

astracmd is a terminal dashboard for monitoring and managing Cesbo Astra instances.

Project:

  https://github.com/unidiag/astracmd

Author:

  Vitali @unidiag Tumasheuski
  info@tva.by

[::b]Dashboard[::-]

The top line shows:

  left    - connection name
  center  - ONLINE / OFFLINE state
  right   - Astra version and commit

The dashboard contains three main panels:

  Adapters - DVB adapters from dvb_tune
  Streams  - streams from make_stream
  Log      - Astra log messages

[::b]Global keys[::-]

  F1        Show this help
  F2        Restart Astra
  F5        Reload config
  F6        Toggle debug log
  F7        Create new adapter or stream
  F8        Delete selected adapter or stream
  F9        Set Astra license
  F10       Quit
  Ctrl+C    Quit
  Esc       Back to the connection list or close dialog

[::b]Dashboard navigation[::-]

  Tab       Switch between Adapters and Streams
  Shift+Tab Switch between Adapters and Streams
  Up/Down   Move selection
  Enter     Edit selected adapter or stream
  E         Edit selected adapter or stream
  Space     Restart selected adapter or stream

[::b]Adapters[::-]

  ALL       Show all streams
  OUTSIDE   Show streams without DVB adapter
  Adapter   Show streams related to selected adapter

Adapter actions:

  Enter     Edit selected adapter
  F7        Create new adapter
  F8        Delete selected adapter
  Space     Restart selected adapter

Adapter editor:

  Enable    Enable or disable adapter
  Name      Adapter name
  Adapter   DVB adapter number
  TP        Transponder string

            S2:12015:R:27500
            S:12015:R:27500
            T:506
            T2:594
            C:410:6875

  LNB       Optional LNB parameters for S/S2 only

            lof1:lof2:slof

            Example:
            10750:10750:10750

            Empty value means Astra default:
            9750:10600:11700

  Mode      Modulation mode

            AUTO means do not send modulation to Astra.

            Supported values:
            QPSK, QAM16, QAM32, QAM64, QAM128, QAM256,
            VSB8, VSB16, PSK8, APSK16, APSK32, DQPSK,
            APSK64, APSK128, APSK256

[::b]Streams[::-]

The Streams panel shows stream names, errors and bitrate.
Bitrate is updated from Astra WebSocket events.

Stream actions:

  Enter     Edit selected stream
  F7        Create new stream
  F8        Delete selected stream
  Space     Restart selected stream

Only SPTS streams are supported by the stream editor.
MPTS streams are not supported in this version of astracmd.

Stream editor:

  Enable    Enable or disable stream
  Name      Stream name
  HbbTV     HbbTV URL (example: http://hbbtv.by/demolink.html)

  Remap     Optional stream remap settings

            Format:
            set_pnr=<pnr>&set_tsid=<tsid>&<map>&filter~=<pids>

            Example:
            set_pnr=160&set_tsid=1&video=192,audio=193&filter~=192,193

  Input     One or more stream inputs

            Examples:
            dvb://a014_a3#pnr=200&cam=a01i
            http://example.com/stream/playlist.m3u8

  Output    One or more stream outputs

            Examples:
            udp://192.168.2.15@239.3.100.45#sync
            srt://vlan1@:5000/?passphrase=12345678

  +         Add input or output line
  -         Remove input or output line
  Ctrl+S    Save dialog
  Esc       Close dialog

When saving a stream, astracmd also sets:

  service_name      - stream name transliterated to Latin if needed
  service_provider  - astracmd version string

[::b]Connection list[::-]

  Enter     Open selected connection
  Left click Open selected connection
  F7        Create new connection
  Space     Edit selected connection
  F8        Delete selected connection
  Esc       Quit

[::b]Log[::-]

The Log panel shows Astra log messages.
Log messages are filtered by selected adapter or selected stream.

[::b]Config file[::-]

Default config path:

  /etc/astra/astracmd.ini

Custom config path can be passed as the first argument:

  ./astracmd /etc/astracmd/conf.ini
`

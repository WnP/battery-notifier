# battery-notifier

**Development Status**: Planning

This is a simple app which notify your laptop battery state, here are the
steps with corresponding notification:

- 100% and charging will notify you to unplug
- 80% and charging will notify you to unplug
- 20% and discharging will notify to plug
- 10% and discharging will notify to plug and will hibernate 1min after

## Dependencies

- [libnotify](https://developer.gnome.org/libnotify/) notify-send
- [zzz](https://github.com/voidlinux/void-runit/blob/master/zzz)
  to manage hibernate

## Install

```
go get github.com/WnP/battery-notifier
```

## Usage

```
echo 'battery-notifier &' >> ~/.xinitrc
```

## Roadmap

- [ ] Units Tests
- [ ] Providing icons
- [ ] OSX compatibility
- [ ] Windows compatibility

Pull requests are welcome.

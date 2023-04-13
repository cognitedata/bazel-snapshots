load(
    "//snapshots/private:snapshots.bzl",
    _change_tracker = "change_tracker",
    _create_tracker_file = "create_tracker_file",
    _snapshots = "snapshots",
)

change_tracker = _change_tracker
create_tracker_file = _create_tracker_file
snapshots = _snapshots

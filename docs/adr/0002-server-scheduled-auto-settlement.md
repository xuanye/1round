# Server-scheduled auto settlement

Automatic settlement is performed by a server-side scheduled task instead of client-side checks or lazy settlement on user access. This keeps the 24-hour inactivity rule reliable even when no participant is online, at the cost of requiring backend scheduling and idempotent settlement processing.

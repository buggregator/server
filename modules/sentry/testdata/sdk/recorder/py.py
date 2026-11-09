"""Records what sentry-python sends: error, message, transaction with spans, logs."""
import time, urllib.request
import sentry_sdk
from sentry_sdk import logger as slog
sentry_sdk.init(dsn="http://publickey@127.0.0.1:18490/42", release="shop@1.2.3", environment="staging",
                server_name="py-01", traces_sample_rate=1.0, enable_logs=True, send_default_pii=True)
class OrderError(Exception): pass
def load_order(order_id): raise OrderError(f"Order {order_id} not found")
sentry_sdk.set_user({"id": "7", "email": "dev@example.com"})
sentry_sdk.set_tag("feature", "checkout")
sentry_sdk.add_breadcrumb(category="cart", message="added item 5", level="info")
try:
    load_order(15)
except OrderError as e:
    sentry_sdk.capture_exception(e)
sentry_sdk.capture_message("Payment gateway slow", level="warning")
with sentry_sdk.start_transaction(name="GET /orders", op="http.server"):
    with sentry_sdk.start_span(op="db.query", name="SELECT * FROM orders WHERE user_id = %s") as span:
        span.set_data("db.system", "postgresql"); time.sleep(0.02)
    try: urllib.request.urlopen("http://127.0.0.1:18490/health", timeout=2)
    except Exception: pass
slog.info("User {user} opened cart", user="7", attributes={"cart_items": 3})
slog.error("Stock sync failed", attributes={"sku": "A-1"})
sentry_sdk.flush(5)

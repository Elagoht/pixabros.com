import { useEffect, useRef } from "react";
import { useLocation } from "react-router-dom";
import { analyticsService } from "@/services/analytics";

const HEARTBEAT_MS = 30_000;

export default function usePageTracking(
  routeName: string,
  enabled = true,
): void {
  const location = useLocation();
  const pageViewIdRef = useRef<string | null>(null);
  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    const pageViewId = crypto.randomUUID();
    pageViewIdRef.current = pageViewId;
    const sessionId = analyticsService.getSessionId();

    // Start
    analyticsService
      .startPageView({
        page_view_id: pageViewId,
        session_id: sessionId,
        route_name: routeName,
        path: location.pathname,
        referrer: document.referrer || undefined,
      })
      .catch(() => {});

    // Heartbeat
    heartbeatRef.current = setInterval(() => {
      analyticsService.heartbeatPageView(pageViewId).catch(() => {});
    }, HEARTBEAT_MS);

    // Cleanup on route change / unmount
    return () => {
      if (heartbeatRef.current) {
        clearInterval(heartbeatRef.current);
        heartbeatRef.current = null;
      }
      const pvid = pageViewIdRef.current;
      if (pvid) {
        analyticsService.endPageView(pvid).catch(() => {});
        pageViewIdRef.current = null;
      }
    };
  }, [routeName, location.pathname, enabled]);
}

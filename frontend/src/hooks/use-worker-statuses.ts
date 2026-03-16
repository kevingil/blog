import { useEffect, useRef, useState } from "react";

import { VITE_WS_URL } from "@/services/constants";
import { getWorkersStatus, type WorkerStatus } from "@/services/workers";

export function useWorkerStatuses() {
  const [workerStatuses, setWorkerStatuses] = useState<
    Record<string, WorkerStatus>
  >({});
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let isMounted = true;

    getWorkersStatus()
      .then((response) => {
        if (!isMounted) {
          return;
        }

        const statuses: Record<string, WorkerStatus> = {};
        response.workers.forEach((worker) => {
          statuses[worker.name] = worker;
        });
        setWorkerStatuses(statuses);
      })
      .catch(console.error);

    return () => {
      isMounted = false;
    };
  }, []);

  useEffect(() => {
    const wsURL =
      VITE_WS_URL ||
      `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/websocket`;
    const ws = new WebSocket(wsURL);
    wsRef.current = ws;

    ws.onopen = () => {
      ws.send(
        JSON.stringify({
          action: "subscribe",
          channel: "worker-status",
        }),
      );
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type !== "worker-status") {
          return;
        }

        setWorkerStatuses((prev) => ({
          ...prev,
          [data.worker_name]: data.status,
        }));
      } catch (error) {
        console.error("Failed to parse WebSocket message:", error);
      }
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };

    return () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(
          JSON.stringify({
            action: "unsubscribe",
            channel: "worker-status",
          }),
        );
      }
      ws.close();
    };
  }, []);

  return workerStatuses;
}

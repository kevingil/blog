import { useCallback, useEffect, useRef, useState } from "react";
import { VITE_API_BASE_URL } from "@/services/constants";

export type ConversationState = "idle" | "connecting" | "live" | "speaking";

const TARGET_RATE = 24000;

type LiveEvent = {
  type?: string;
  model?: string;
  text?: string;
  delta?: string;
  role?: string;
  audio?: string;
  sampleRate?: number;
  requestId?: string;
  message?: string;
  error?: string;
};

function liveSocketUrl(): string {
  return `${VITE_API_BASE_URL.replace(/^http/, "ws")}/agent/live`;
}

function downsample(buffer: Float32Array, fromRate: number, toRate: number): Float32Array {
  if (fromRate === toRate) {
    return buffer;
  }
  const ratio = fromRate / toRate;
  const length = Math.round(buffer.length / ratio);
  const result = new Float32Array(length);
  for (let index = 0; index < length; index += 1) {
    const start = Math.floor(index * ratio);
    const end = Math.min(buffer.length, Math.floor((index + 1) * ratio));
    let sum = 0;
    for (let sample = start; sample < end; sample += 1) {
      sum += buffer[sample] ?? 0;
    }
    result[index] = sum / Math.max(1, end - start);
  }
  return result;
}

function pcm16Base64(samples: Float32Array): string {
  const bytes = new Uint8Array(samples.length * 2);
  const view = new DataView(bytes.buffer);
  for (let index = 0; index < samples.length; index += 1) {
    const sample = Math.max(-1, Math.min(1, samples[index] ?? 0));
    view.setInt16(index * 2, sample < 0 ? sample * 0x8000 : sample * 0x7fff, true);
  }
  let binary = "";
  const chunk = 0x2000;
  for (let index = 0; index < bytes.length; index += chunk) {
    binary += String.fromCharCode(...bytes.subarray(index, index + chunk));
  }
  return btoa(binary);
}

function playPcm(
  context: AudioContext,
  nextTime: { current: number },
  audioBase64: string,
  sampleRate: number,
) {
  const binary = atob(audioBase64);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  const view = new DataView(bytes.buffer);
  const samples = new Float32Array(Math.floor(bytes.length / 2));
  for (let index = 0; index < samples.length; index += 1) {
    samples[index] = view.getInt16(index * 2, true) / 32768;
  }
  if (samples.length === 0) {
    return;
  }
  const buffer = context.createBuffer(1, samples.length, sampleRate);
  buffer.getChannelData(0).set(samples);
  const source = context.createBufferSource();
  source.buffer = buffer;
  source.connect(context.destination);
  const startAt = Math.max(context.currentTime, nextTime.current);
  source.start(startAt);
  nextTime.current = startAt + buffer.duration;
}

export function useConversation(options: {
  enabled: boolean;
  articleId?: string;
  getDocument: () => { content: string; markdown: string };
  onUserTranscript: (text: string) => void;
  onDelegation: (requestId: string, message: string) => void;
}) {
  const { enabled, articleId } = options;
  const [state, setState] = useState<ConversationState>("idle");
  const [error, setError] = useState<string | null>(null);
  const [caption, setCaption] = useState("");
  const optionsRef = useRef(options);
  optionsRef.current = options;
  const socketRef = useRef<WebSocket | null>(null);

  const sendText = useCallback((text: string) => {
    const socket = socketRef.current;
    const trimmed = text.trim();
    if (!trimmed || !socket || socket.readyState !== WebSocket.OPEN) {
      setError("Live session is not connected");
      return;
    }
    socket.send(JSON.stringify({ type: "text", text: trimmed }));
  }, []);

  useEffect(() => {
    if (!enabled || !articleId) {
      setState("idle");
      setCaption("");
      return;
    }

    let stopped = false;
    let context: AudioContext | null = null;
    let stream: MediaStream | null = null;
    let processor: ScriptProcessorNode | null = null;
    let contextTimer = 0;
    const nextTime = { current: 0 };
    const socket = new WebSocket(liveSocketUrl());
    socketRef.current = socket;
    setError(null);
    setCaption("");
    setState("connecting");

    const sendContext = () => {
      if (socket.readyState !== WebSocket.OPEN) {
        return;
      }
      const document = optionsRef.current.getDocument();
      socket.send(JSON.stringify({
        type: "context",
        documentContent: document.content,
        documentMarkdown: document.markdown,
      }));
    };

    const startMic = async () => {
      try {
        stream = await navigator.mediaDevices.getUserMedia({
          audio: { echoCancellation: true, noiseSuppression: true, channelCount: 1 },
        });
        if (stopped) {
          stream.getTracks().forEach((track) => track.stop());
          return;
        }
        context = new AudioContext();
        await context.resume();
        const source = context.createMediaStreamSource(stream);
        processor = context.createScriptProcessor(4096, 1, 1);
        processor.onaudioprocess = (event) => {
          if (!context || socket.readyState !== WebSocket.OPEN) {
            return;
          }
          const input = event.inputBuffer.getChannelData(0);
          const audio = pcm16Base64(downsample(input, context.sampleRate, TARGET_RATE));
          if (audio) {
            socket.send(JSON.stringify({ type: "audio", audio }));
          }
        };
        const silent = context.createGain();
        silent.gain.value = 0;
        source.connect(processor);
        processor.connect(silent);
        silent.connect(context.destination);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Microphone permission denied");
      }
    };

    socket.onopen = () => {
      const document = optionsRef.current.getDocument();
      socket.send(JSON.stringify({
        type: "start",
        articleId,
        documentContent: document.content,
        documentMarkdown: document.markdown,
      }));
      void startMic();
    };

    socket.onmessage = (event) => {
      let message: LiveEvent;
      try {
        message = JSON.parse(String(event.data)) as LiveEvent;
      } catch {
        return;
      }
      if (message.type === "started") {
        setState("live");
        contextTimer = window.setInterval(sendContext, 2000);
        return;
      }
      if (message.type === "transcript" && message.role === "user" && message.text) {
        setCaption("");
        optionsRef.current.onUserTranscript(message.text);
        return;
      }
      if (message.type === "transcript" && message.role === "assistant" && message.delta) {
        setCaption((current) => `${current}${message.delta}`);
        return;
      }
      if (message.type === "audio" && message.audio && context) {
        setState("speaking");
        playPcm(context, nextTime, message.audio, message.sampleRate || TARGET_RATE);
        window.setTimeout(() => {
          if (!stopped && nextTime.current <= (context?.currentTime ?? 0)) {
            setState("live");
          }
        }, 400);
        return;
      }
      if (message.type === "delegation" && message.requestId) {
        setCaption("");
        optionsRef.current.onDelegation(message.requestId, message.message || "");
        return;
      }
      if (message.type === "error") {
        setError(message.error || "Live session failed");
      }
    };

    socket.onerror = () => {
      setError("Could not connect to GPT-Live");
      setState("idle");
    };

    return () => {
      stopped = true;
      window.clearInterval(contextTimer);
      if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: "stop" }));
      }
      socket.close();
      socketRef.current = null;
      processor?.disconnect();
      stream?.getTracks().forEach((track) => track.stop());
      void context?.close();
      setState("idle");
    };
  }, [articleId, enabled]);

  return { state, error, caption, sendText };
}

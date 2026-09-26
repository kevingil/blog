import { useCallback, useEffect, useRef, useState } from "react";

export type ConversationState = "idle" | "listening" | "recording" | "speaking";

const SILENCE_MS = 900;
const SPEECH_THRESHOLD = 0.018;

function encodeBlob(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onloadend = () => {
      const result = String(reader.result || "");
      const comma = result.indexOf(",");
      resolve(comma >= 0 ? result.slice(comma + 1) : result);
    };
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(blob);
  });
}

export function useConversation(options: {
  enabled: boolean;
  busy?: boolean;
  onUtterance: (audioBase64: string, mimeType: string) => Promise<void> | void;
}) {
  const { enabled, busy = false, onUtterance } = options;
  const [state, setState] = useState<ConversationState>("idle");
  const [error, setError] = useState<string | null>(null);
  const mediaRef = useRef<MediaStream | null>(null);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const speakingRef = useRef(false);
  const lastVoiceRef = useRef(0);
  const rafRef = useRef<number>(0);
  const discardRef = useRef(false);
  const pausedRef = useRef(false);
  const onUtteranceRef = useRef(onUtterance);
  onUtteranceRef.current = onUtterance;
  pausedRef.current = busy;

  const stopTracks = useCallback(() => {
    cancelAnimationFrame(rafRef.current);
    recorderRef.current?.state === "recording" && recorderRef.current.stop();
    recorderRef.current = null;
    mediaRef.current?.getTracks().forEach((track) => track.stop());
    mediaRef.current = null;
    analyserRef.current = null;
    void audioContextRef.current?.close();
    audioContextRef.current = null;
    setState("idle");
  }, []);

  const flushUtterance = useCallback(async () => {
    const chunks = chunksRef.current;
    chunksRef.current = [];
    if (discardRef.current) {
      discardRef.current = false;
      return;
    }
    if (pausedRef.current || chunks.length === 0) {
      return;
    }
    const mimeType = recorderRef.current?.mimeType || "audio/webm";
    const blob = new Blob(chunks, { type: mimeType });
    if (blob.size < 256) {
      return;
    }
    const audioBase64 = await encodeBlob(blob);
    await onUtteranceRef.current(audioBase64, mimeType);
  }, []);

  const monitor = useCallback(() => {
    const analyser = analyserRef.current;
    if (!analyser) {
      return;
    }
    if (pausedRef.current) {
      speakingRef.current = false;
      rafRef.current = requestAnimationFrame(monitor);
      return;
    }
    const data = new Uint8Array(analyser.fftSize);
    analyser.getByteTimeDomainData(data);
    let total = 0;
    for (const sample of data) {
      const centered = (sample - 128) / 128;
      total += centered * centered;
    }
    const rms = Math.sqrt(total / data.length);
    const now = performance.now();
    if (rms > SPEECH_THRESHOLD) {
      lastVoiceRef.current = now;
      if (!speakingRef.current) {
        speakingRef.current = true;
        setState("recording");
      }
    } else if (speakingRef.current && now - lastVoiceRef.current > SILENCE_MS) {
      speakingRef.current = false;
      setState("listening");
      if (recorderRef.current?.state === "recording") {
        recorderRef.current.stop();
      }
    }
    rafRef.current = requestAnimationFrame(monitor);
  }, []);

  const start = useCallback(async () => {
    setError(null);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      mediaRef.current = stream;
      const context = new AudioContext();
      audioContextRef.current = context;
      const source = context.createMediaStreamSource(stream);
      const analyser = context.createAnalyser();
      analyser.fftSize = 2048;
      source.connect(analyser);
      analyserRef.current = analyser;
      const mimeType = MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
        ? "audio/webm;codecs=opus"
        : "audio/webm";
      const startRecorder = () => {
        const recorder = new MediaRecorder(stream, { mimeType });
        recorderRef.current = recorder;
        chunksRef.current = [];
        recorder.ondataavailable = (event) => {
          if (event.data.size > 0) {
            chunksRef.current.push(event.data);
          }
        };
        recorder.onstop = () => {
          void flushUtterance().then(() => {
            if (mediaRef.current && enabled) {
              startRecorder();
            }
          });
        };
        recorder.start();
      };
      startRecorder();
      speakingRef.current = false;
      lastVoiceRef.current = performance.now();
      setState("listening");
      rafRef.current = requestAnimationFrame(monitor);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Microphone permission denied");
      stopTracks();
    }
  }, [enabled, flushUtterance, monitor, stopTracks]);

  useEffect(() => {
    if (enabled) {
      void start();
    } else {
      stopTracks();
    }
    return () => {
      stopTracks();
    };
  }, [enabled, start, stopTracks]);

  const playSpeech = useCallback(async (audioBase64: string, mimeType = "audio/wav") => {
    if (!audioBase64) {
      return;
    }
    pausedRef.current = true;
    discardRef.current = true;
    speakingRef.current = false;
    chunksRef.current = [];
    if (recorderRef.current?.state === "recording") {
      recorderRef.current.stop();
    }
    setState("speaking");
    const audio = new Audio(`data:${mimeType};base64,${audioBase64}`);
    await new Promise<void>((resolve) => {
      audio.onended = () => resolve();
      audio.onerror = () => resolve();
      void audio.play().catch(() => resolve());
    });
    pausedRef.current = busy;
    if (enabled) {
      setState("listening");
    } else {
      setState("idle");
    }
  }, [busy, enabled]);

  return { state, error, playSpeech };
}

import { useEffect, useMemo, useRef, useState } from "react";

import { MatchDetailResponse, OperatorApiClient, ResultListItem } from "../../api";
import { defaultBaseUrl, EnqueueState, isAbortError, LoadState, messageOf, normalizeBaseUrl } from "./operatorPageSupport";

const ACTIVE_POLL_MS = 5_000;
const COMPLETED_POLL_MS = 10_000;
const DETAIL_POLL_MS = 15_000;

export function useOperatorPageState() {
  const [baseUrl, setBaseUrl] = useState(() => defaultBaseUrl());
  const client = useMemo(() => new OperatorApiClient(normalizeBaseUrl(baseUrl)), [baseUrl]);

  const [activeItems, setActiveItems] = useState<ResultListItem[]>([]);
  const [completedItems, setCompletedItems] = useState<ResultListItem[]>([]);
  const [activeState, setActiveState] = useState<LoadState>("loading");
  const [completedState, setCompletedState] = useState<LoadState>("loading");
  const [detailState, setDetailState] = useState<LoadState>("idle");
  const [enqueueState, setEnqueueState] = useState<EnqueueState>("idle");
  const [activeError, setActiveError] = useState<string>();
  const [completedError, setCompletedError] = useState<string>();
  const [detailError, setDetailError] = useState<string>();
  const [enqueueError, setEnqueueError] = useState<string>();
  const [selectedSubmissionId, setSelectedSubmissionId] = useState<string>();
  const [detail, setDetail] = useState<MatchDetailResponse>();
  const [detailReloadToken, setDetailReloadToken] = useState(0);
  const detailRequestSequence = useRef(0);

  useEffect(() => {
    let canceled = false;

    const load = async () => {
      setActiveState((current) => (current === "ready" ? current : "loading"));
      try {
        const items = await client.listActive();
        if (canceled) {
          return;
        }
        setActiveItems(items);
        setActiveState("ready");
        setActiveError(undefined);
      } catch (error) {
        if (canceled) {
          return;
        }
        setActiveState("error");
        setActiveError(messageOf(error));
      }
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, ACTIVE_POLL_MS);
    return () => {
      canceled = true;
      window.clearInterval(timer);
    };
  }, [client]);

  useEffect(() => {
    let canceled = false;

    const load = async () => {
      setCompletedState((current) => (current === "ready" ? current : "loading"));
      try {
        const items = await client.listCompleted();
        if (canceled) {
          return;
        }
        setCompletedItems(items);
        setCompletedState("ready");
        setCompletedError(undefined);
        setSelectedSubmissionId((current) => current ?? items[0]?.submission_id);
      } catch (error) {
        if (canceled) {
          return;
        }
        setCompletedState("error");
        setCompletedError(messageOf(error));
      }
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, COMPLETED_POLL_MS);
    return () => {
      canceled = true;
      window.clearInterval(timer);
    };
  }, [client]);

  useEffect(() => {
    if (!selectedSubmissionId) {
      setDetail(undefined);
      setDetailState("idle");
      setDetailError(undefined);
      return;
    }

    let canceled = false;
    let inFlightController: AbortController | undefined;

    const load = async () => {
      inFlightController?.abort();
      const controller = new AbortController();
      const requestSequence = detailRequestSequence.current + 1;
      detailRequestSequence.current = requestSequence;
      inFlightController = controller;
      setDetailState((current) => (current === "ready" ? current : "loading"));
      try {
        const response = await client.getMatchDetail(selectedSubmissionId, controller.signal);
        if (canceled || requestSequence !== detailRequestSequence.current) {
          return;
        }
        setDetail(response);
        setDetailState("ready");
        setDetailError(undefined);
      } catch (error) {
        if (canceled || isAbortError(error) || requestSequence !== detailRequestSequence.current) {
          return;
        }
        setDetailState("error");
        setDetailError(messageOf(error));
      }
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, DETAIL_POLL_MS);
    return () => {
      canceled = true;
      inFlightController?.abort();
      window.clearInterval(timer);
    };
  }, [client, detailReloadToken, selectedSubmissionId]);

  const handleEnqueue = async (presetId: string) => {
    setEnqueueState("submitting");
    setEnqueueError(undefined);
    try {
      await client.enqueuePreset(presetId);
      setEnqueueState("success");
      const items = await client.listActive();
      setActiveItems(items);
      setActiveState("ready");
      setActiveError(undefined);
    } catch (error) {
      setEnqueueState("error");
      setEnqueueError(messageOf(error));
    }
  };

  return {
    baseUrl,
    setBaseUrl,
    activeItems,
    completedItems,
    activeState,
    completedState,
    detailState,
    enqueueState,
    activeError,
    completedError,
    detailError,
    enqueueError,
    selectedSubmissionId,
    setSelectedSubmissionId,
    detail,
    reloadDetail: () => setDetailReloadToken((current) => current + 1),
    enqueuePreset: handleEnqueue,
  };
}

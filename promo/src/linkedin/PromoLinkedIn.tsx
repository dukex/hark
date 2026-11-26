import { linearTiming, TransitionSeries } from "@remotion/transitions";
import { fade } from "@remotion/transitions/fade";
import { AbsoluteFill, useVideoConfig } from "remotion";
import { Converge } from "./Converge";
import { Flow } from "./Flow";
import { Install } from "./Install";
import { Tabs } from "./Tabs";

export const PromoLinkedIn = () => {
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill style={{ backgroundColor: "#0D0D0D" }}>
      <TransitionSeries>
        <TransitionSeries.Sequence name="Seven tabs" durationInFrames={8 * fps}>
          <Tabs />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence
          name="GitHub to Claude"
          durationInFrames={11.5 * fps}
        >
          <Flow />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence
          name="Every service"
          durationInFrames={7 * fps}
        >
          <Converge />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence name="Install" durationInFrames={5 * fps}>
          <Install />
        </TransitionSeries.Sequence>
      </TransitionSeries>
    </AbsoluteFill>
  );
};

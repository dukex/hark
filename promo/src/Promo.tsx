import { linearTiming, TransitionSeries } from "@remotion/transitions";
import { fade } from "@remotion/transitions/fade";
import { AbsoluteFill, useVideoConfig } from "remotion";
import { AgentSession } from "./scenes/AgentSession";
import { AskHear } from "./scenes/AskHear";
import { Hook } from "./scenes/Hook";
import { Install } from "./scenes/Install";
import { OneLine } from "./scenes/OneLine";
import { WithoutHark } from "./scenes/WithoutHark";

export const Promo = () => {
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill style={{ backgroundColor: "#0D0D0D" }}>
      <TransitionSeries>
        <TransitionSeries.Sequence name="Hook" durationInFrames={6.5 * fps}>
          <Hook />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence
          name="Without hark"
          durationInFrames={10.5 * fps}
        >
          <WithoutHark />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence
          name="GitHub review"
          durationInFrames={11 * fps}
        >
          <AgentSession
            source="hark github"
            kind="pr.changes_requested"
            detail="acme/checkout#15 @carol"
            reply="Carol wants the migration in its own PR. I can move it to a new branch and open that PR. Want me to?"
          />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence
          name="Datadog alert"
          durationInFrames={10 * fps}
        >
          <AgentSession
            source="hark datadog"
            kind="datadog.alert"
            detail='service:checkout "p95 latency above 800ms"'
            reply="The checkout latency alert just fired. I'll finish the refund change unless you want me on the alert first."
          />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence
          name="One line per event"
          durationInFrames={8.5 * fps}
        >
          <OneLine />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence
          name="Ask and hear"
          durationInFrames={8.5 * fps}
        >
          <AskHear />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence name="Install" durationInFrames={8 * fps}>
          <Install />
        </TransitionSeries.Sequence>
      </TransitionSeries>
    </AbsoluteFill>
  );
};

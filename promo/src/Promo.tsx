import { linearTiming, TransitionSeries } from "@remotion/transitions";
import { fade } from "@remotion/transitions/fade";
import { AbsoluteFill, useVideoConfig } from "remotion";
import { Chaos } from "./scenes/Chaos";
import { Converge } from "./scenes/Converge";
import { EventFlow } from "./scenes/EventFlow";
import { Hook } from "./scenes/Hook";
import { Install } from "./scenes/Install";

export const Promo = () => {
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill style={{ backgroundColor: "#0D0D0D" }}>
      <TransitionSeries>
        <TransitionSeries.Sequence
          name="Seven tabs"
          durationInFrames={10.5 * fps}
        >
          <Chaos />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence name="Hook" durationInFrames={5 * fps}>
          <Hook />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence
          name="GitHub review"
          durationInFrames={11 * fps}
        >
          <EventFlow
            provider="GitHub"
            event="Carol requested changes on #15"
            command="hark github"
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
          name="Slack mention"
          durationInFrames={10 * fps}
        >
          <EventFlow
            provider="Slack"
            event="Ana in #deploys: can you look at the rollout?"
            command="hark slack"
            kind="slack.mentioned"
            detail="#deploys @ana"
            reply="Ana needs you in #deploys to look at the rollout. I won't answer in Slack for you."
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
          <EventFlow
            provider="Datadog"
            event="p95 latency above 800ms on checkout"
            command="hark datadog"
            kind="datadog.alert"
            detail="service:checkout"
            reply="The checkout latency alert just fired. I'll finish the refund change unless you want me on the alert first."
          />
        </TransitionSeries.Sequence>
        <TransitionSeries.Transition
          presentation={fade()}
          timing={linearTiming({ durationInFrames: 0.5 * fps })}
        />
        <TransitionSeries.Sequence
          name="Every service"
          durationInFrames={8.5 * fps}
        >
          <Converge />
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

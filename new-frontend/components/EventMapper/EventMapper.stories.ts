import { Meta, Story } from "@storybook/vue3";
import EventMapper from '~/components/EventMapper/EventMapper.vue';
import monologEventMock from '~/mocks/monolog.json'
import sentryEventMock from '~/mocks/sentry-spiral.json'
import smtpEventMock from '~/mocks/smtp-welcome.json'
import varDumpEventMock from '~/mocks/var-dump-object.json'
import profilerEventMock from '~/mocks/profiler.json'
import inspectorEventMock from '~/mocks/inspector.json'

export default {
  title: "Event/EventMapper",
  component: EventMapper
} as Meta<typeof EventMapper>;

const Template: Story = (args) => ({
  components: { EventMapper },
  setup() {
    return {
      args,
    };
  },
  template: `<event-mapper v-bind="args" />`,
});

export const Monolog = Template.bind({});

Monolog.args = {
  event: monologEventMock,
};

export const Sentry = Template.bind({});

Sentry.args = {
  event: sentryEventMock,
};

export const Smtp = Template.bind({});

Smtp.args = {
  event: smtpEventMock,
};

export const VarDump = Template.bind({});

VarDump.args = {
  event: varDumpEventMock,
};

export const Profiler = Template.bind({});

Profiler.args = {
  event: profilerEventMock,
};

export const Inspector = Template.bind({});

Inspector.args = {
  event: inspectorEventMock,
};

export const Unknown = Template.bind({});

Unknown.args = {
  event: { ...smtpEventMock, type: 'unknown' },
};

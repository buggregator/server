import { Meta, Story } from "@storybook/vue3";
import { normalizeVarDumpEvent } from "~/utils/normalize-event";
import EventVarDump from '~/components/EventVarDump/EventVarDump.vue';

import varDumpObjectEventMock from '~/mocks/var-dump-object.json'
import varDumpNumberEventMock from '~/mocks/var-dump-number.json'
import varDumpStringEventMock from '~/mocks/var-dump-string.json'
import varDumpArrayEventMock from '~/mocks/var-dump-array.json'

export default {
  title: "VarDump",
  component: EventVarDump
} as Meta<typeof EventVarDump>;

const Template: Story = (args) => ({
  components: { EventVarDump },
  setup() {
    return {
      args,
    };
  },
  template: `<event-var-dump v-bind="args" />`,
});

export const Object = Template.bind({});

Object.args = {
  event: normalizeVarDumpEvent(varDumpObjectEventMock),
};

export const Number = Template.bind({});

Number.args = {
  event: normalizeVarDumpEvent(varDumpNumberEventMock),
};

export const String = Template.bind({});

String.args = {
  event: normalizeVarDumpEvent(varDumpStringEventMock),
};

export const Array = Template.bind({});

Array.args = {
  event: normalizeVarDumpEvent(varDumpArrayEventMock),
};

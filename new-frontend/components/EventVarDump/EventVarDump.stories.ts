import { Meta, Story } from "@storybook/vue3";
import { normalizeVarDumpEvent } from "~/utils/normalize-event";
import EventVarDump from '~/components/EventVarDump/EventVarDump.vue';
import varDumpEventMock from '~/mocks/var-dump.json'

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

export const Event = Template.bind({});

Event.args = {
  event: normalizeVarDumpEvent(varDumpEventMock),
};

import { Meta, Story } from "@storybook/vue3";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";

export default {
  title: "Components/EventTableRow",
  component: EventTableRow
} as Meta<typeof EventTableRow>;

const Template: Story = (args) => ({
  components: { EventTableRow },
  setup() {
    return {
      args,
    };
  },
  template: `<EventTableRow :title="args.title">
  This is a row 1
  </EventTableRow>`,
});

export const TableRow = Template.bind({});
TableRow.args = {
  title: "Row 1",
};

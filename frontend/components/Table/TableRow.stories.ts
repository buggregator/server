import { Meta, Story } from "@storybook/vue3";
import TableRowComponent from "./TableRow.vue";

export default {
  title: "Components/Table",
  component: TableRowComponent
} as Meta<typeof TableRowComponent>;

const Template: Story = (args) => ({
  components: { TableRowComponent },
  setup() {
    return {
      args,
    };
  },
  template: `<TableRowComponent :title="args.title">
  This is a row 1
  </TableRowComponent>`,
});

export const TableRow = Template.bind({});
TableRow.args = {
  title: "Row 1",
};
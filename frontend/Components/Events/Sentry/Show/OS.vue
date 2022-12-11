<template>
    <section class="py-5 px-4 md:px-6 lg:px-8 border-b" v-if="event.contexts.os.name">
        <h3 class="text-muted font-bold uppercase text-sm mb-5">operating system</h3>

        <Table>
            <TableRow v-if="event.contexts.os.kernel_version" :title="'Kernel Version'">
                {{ event.contexts.os.kernel_version }}
            </TableRow>

            <TableRow v-if="event.contexts.os.name" :title="'Name'">
                {{ event.contexts.os.name }}
            </TableRow>

            <TableRow v-if="event.contexts.os.rooted" :title="'Rooted'">
                {{ event.contexts.os.rooted }}
            </TableRow>

            <TableRow v-if="event.contexts.os.version" :title="'Version'">
                {{ event.contexts.os.version }}
            </TableRow>
        </Table>
    </section>
</template>

<script>
import Table from "@/Components/UI/Table";
import TableRow from "@/Components/UI/TableRow";
export default {
    components: {
        Table, TableRow
    },
    props: {
        event: Object
    }
}
</script>

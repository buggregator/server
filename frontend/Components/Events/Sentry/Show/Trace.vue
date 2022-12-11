<template>
    <section class="py-5 px-4 md:px-6 lg:px-8 border-b">
        <h3 class="text-muted font-bold uppercase text-sm mb-5">trace details</h3>

        <Table>
            <TableRow v-if="event.contexts.trace.status" :title="'Status'">
                {{ event.contexts.trace.status }}
            </TableRow>

            <TableRow v-if="event.contexts.trace.trace_id" :title="'Trace ID'">
                {{ event.contexts.trace.trace_id }}
            </TableRow>

            <TableRow v-if="event.contexts.trace.span_id" :title="'Span ID'">
                {{ event.contexts.trace.span_id }}
            </TableRow>

            <TableRow v-if="event.contexts.trace.op" :title="'Operation Name'">
                {{ event.contexts.trace.op }}
            </TableRow>

            <TableRow v-if="event.contexts.trace.data" :title="'Data'">
                <CodeSnippet class="mt-3" language="json" :code="event.contexts.trace.data"/>
            </TableRow>

            <TableRow v-if="event.contexts.trace.tags" :title="'Tags'">
                <CodeSnippet class="mt-3" language="json" :code="event.contexts.trace.tags"/>
            </TableRow>
        </Table>
    </section>
</template>

<script>
import Table from "@/Components/UI/Table";
import TableRow from "@/Components/UI/TableRow";
import CodeSnippet from "@/Components/UI/CodeSnippet";

export default {
    components: {
        Table, TableRow, CodeSnippet
    },
    props: {
        event: Object
    }
}
</script>

package eu.epos.semanticaggregator.grpc;

import io.grpc.Context;
import io.grpc.Contexts;
import io.grpc.Metadata;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import io.grpc.ServerInterceptor;

/** Carries the identity supplied by the trusted gateway to the import handler. */
public final class ActorInterceptor implements ServerInterceptor {
    static final Context.Key<String> ACTOR = Context.key("actor");
    private static final Metadata.Key<String> ACTOR_HEADER =
            Metadata.Key.of("x-epos-user-id", Metadata.ASCII_STRING_MARSHALLER);

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call, Metadata headers, ServerCallHandler<ReqT, RespT> next) {
        String actor = headers.get(ACTOR_HEADER);
        Context context = actor == null ? Context.current() : Context.current().withValue(ACTOR, actor);
        return Contexts.interceptCall(context, call, headers, next);
    }
}

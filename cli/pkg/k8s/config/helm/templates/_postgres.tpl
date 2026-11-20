{{- define "epos.postgres" -}}
{{- if .database.enabled }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .name }}
  labels:
    {{- include "epos.labels" .root | nindent 4 }}
    app.kubernetes.io/component: {{ .name }}
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      {{- include "epos.selectorLabels" .root | nindent 6 }}
      app.kubernetes.io/component: {{ .name }}
  template:
    metadata:
      labels:
        {{- include "epos.selectorLabels" .root | nindent 8 }}
        app.kubernetes.io/component: {{ .name }}
      annotations:
        checksum/secret-{{ .name }}: {{ .database.password | toString | sha256sum }}
    spec:
      {{- include "epos.image_pull_secrets" .root | nindent 6 }}
      containers:
        - name: {{ .name }}
          image: {{ .image | quote }}
          imagePullPolicy: {{ .root.Values.image_pull_policy }}
          env:
            - name: POSTGRES_USER
              value: {{ .database.user | quote }}
            - name: POSTGRES_DB
              value: {{ .database.db_name | quote }}
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ .name }}
                  key: password
          readinessProbe:
            exec:
              command: ["sh", "-c", "pg_isready -U \"$POSTGRES_USER\" -d \"$POSTGRES_DB\""]
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: {{ .name }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .name }}
  labels:
    {{- include "epos.labels" .root | nindent 4 }}
    app.kubernetes.io/component: {{ .name }}
spec:
  ports:
    - port: 5432
  selector:
    {{- include "epos.selectorLabels" .root | nindent 4 }}
    app.kubernetes.io/component: {{ .name }}
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ .name }}
  labels:
    {{- include "epos.labels" .root | nindent 4 }}
    app.kubernetes.io/component: {{ .name }}
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 5Gi
{{- end }}
{{- end -}}
